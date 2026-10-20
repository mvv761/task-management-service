package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "task-management-service/docs"
	rediscache "task-management-service/internal/cache/redis"
	"task-management-service/internal/config"
	"task-management-service/internal/http/handler"
	appmiddleware "task-management-service/internal/http/middleware"
	mysqlrepo "task-management-service/internal/repository/mysql"
	"task-management-service/internal/service"
	"github.com/go-chi/chi/v5"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	goredis "github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Task Management Service API
// @version 1.0
// @description HTTP API сервиса задач внутри команд: пользователи, роли, задачи, комментарии, история и аналитика.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Заголовок Authorization: Bearer <JWT>
func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	)
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	dbConfig := mysqldriver.Config{
		User:                 cfg.MySQL.User,
		Passwd:               cfg.MySQL.Password,
		Net:                  "tcp",
		Addr:                 cfg.MySQL.Host + ":" + cfg.MySQL.Port,
		DBName:               cfg.MySQL.Database,
		ParseTime:            true,
		AllowNativePasswords: true,
	}

	db, err := sqlx.Open("mysql", dbConfig.FormatDSN())
	if err != nil {
		logger.Error("open database failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error("ping database failed", "error", err)
		os.Exit(1)
	}

	userRepository := mysqlrepo.NewUserRepository(db)

	authService := service.NewAuthService(userRepository, cfg.JWT.Secret, time.Duration(cfg.JWT.TTLMinutes)*time.Minute)

	authHandler := handler.NewAuthHandler(authService)

	authMiddleware := appmiddleware.NewAuth(cfg.JWT.Secret)
	publicRateLimiter, err := appmiddleware.NewRateLimiter(cfg.RateLimit.RPS, cfg.RateLimit.Burst)
	if err != nil {
		logger.Error("create public rate limiter failed", "error", err)
		os.Exit(1)
	}
	defer publicRateLimiter.Close()

	authenticatedRateLimiter, err := appmiddleware.NewRateLimiter(cfg.RateLimit.RPS, cfg.RateLimit.Burst)
	if err != nil {
		logger.Error("create authenticated rate limiter failed", "error", err)
		os.Exit(1)
	}
	defer authenticatedRateLimiter.Close()

	teamRepository := mysqlrepo.NewTeamRepository(db)

	teamService := service.NewTeamService(teamRepository, userRepository)

	teamHandler := handler.NewTeamHandler(teamService)

	taskRepository := mysqlrepo.NewTaskRepository(db)

	redisClient := goredis.NewClient(&goredis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
	})

	taskCache := rediscache.NewTaskCache(redisClient)

	taskService := service.NewTaskService(taskRepository,
		teamRepository, taskCache, logger)

	taskHandler := handler.NewTaskHandler(taskService)

	taskHistoryRepository := mysqlrepo.NewTaskHistoryRepository(db)

	taskHistoryService := service.NewTaskHistoryService(taskHistoryRepository, taskRepository, teamRepository)

	taskHistoryHandler := handler.NewTaskHistoryHandler(taskHistoryService)

	taskCommentRepository := mysqlrepo.NewTaskCommentRepository(db)

	taskCommentService := service.NewTaskCommentService(taskCommentRepository, taskRepository, teamRepository)

	taskCommentHandler := handler.NewTaskCommentHandler(taskCommentService)

	teamStatsRepository := mysqlrepo.NewTeamStatsRepository(db)

	teamStatsService := service.NewTeamStatsService(teamStatsRepository, teamRepository)

	teamStatsHandler := handler.NewTeamStatsHandler(teamStatsService)

	r := chi.NewRouter()
	r.Use(appmiddleware.RequestID)
	r.Use(appmiddleware.RequestLogging(logger))
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.PersistAuthorization(true),
		httpSwagger.UIConfig(map[string]string{
			"requestInterceptor": `function(request) {
				const authorization = request.headers.Authorization;
				if (authorization && !authorization.startsWith("Bearer ")) {
					request.headers.Authorization = "Bearer " + authorization;
				}
				return request;
			}`,
		}),
	))
	r.Get("/health", handler.Health)

	r.Group(func(r chi.Router) {
		r.Use(publicRateLimiter.Middleware)
		r.Post("/api/v1/register", authHandler.Register)
		r.Post("/api/v1/login", authHandler.Login)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(authenticatedRateLimiter.Middleware)

		r.Post("/api/v1/teams", teamHandler.CreateTeam)
		r.Get("/api/v1/teams", teamHandler.ListTeams)
		r.Post("/api/v1/teams/{id}/invite", teamHandler.InviteMember)
		r.Patch("/api/v1/teams/{team_id}/members/{user_id}", teamHandler.UpdateMemberRole)

		r.Post("/api/v1/tasks", taskHandler.CreateTask)
		r.Get("/api/v1/tasks", taskHandler.ListTasks)
		r.Get("/api/v1/tasks/{id}", taskHandler.GetTask)
		r.Put("/api/v1/tasks/{id}", taskHandler.UpdateTask)

		r.Get("/api/v1/tasks/{id}/history", taskHistoryHandler.ListTaskHistory)

		r.Post("/api/v1/tasks/{id}/comments", taskCommentHandler.CreateComment)
		r.Get("/api/v1/tasks/{id}/comments", taskCommentHandler.ListComments)

		r.Get("/api/v1/teams/{team_id}/stats", teamStatsHandler.GetTeamStats)
	})

	server := &http.Server{
		Addr:    ":" + cfg.HTTP.Port,
		Handler: r,
	}

	serverCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server starting", "port", cfg.HTTP.Port)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	case <-serverCtx.Done():
		logger.Info("http server shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown failed", "error", err)
			os.Exit(1)
		}

		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed during shutdown", "error", err)
			os.Exit(1)
		}

		logger.Info("http server stopped")
	}
}
