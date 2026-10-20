# Task Management Service

Бэкенд для командной работы с задачами: роли, история правок, комментарии и простой SQL-отчёт.

Что есть в API:

- регистрация / логин (JWT);
- команды и роли `owner`, `admin`, `member`;
- CRUD задач с фильтрами и пагинацией;
- optimistic locking по `version`;
- история изменений и комментарии;
- кеш списка задач в Redis;
- rate limit на HTTP;
- статистика по команде.

---

## Stack

- Go
- Chi
- MySQL 8
- Redis
- `database/sql` + `sqlx`
- `golang-migrate`
- JWT
- bcrypt
- Docker Compose
- Swagger / Swaggo
- `golang.org/x/time/rate`

---

## Architecture

Приложение разделено на несколько слоёв:

```text
HTTP handlers
      ↓
   Services
      ↓
Repositories / Cache
      ↓
 MySQL / Redis
```

Структура проекта:

```text
cmd/
  api/                    # entrypoint и dependency wiring

internal/
  config/                 # конфигурация приложения из environment variables
  domain/                 # domain entities и ошибки
  policy/                 # бизнес-правила и проверки прав доступа

  repository/
    mysql/                # MySQL repositories

  cache/
    redis/                # Redis cache implementation

  service/                # бизнес-логика приложения

  http/
    handler/              # HTTP handlers
    middleware/           # JWT и другие middleware
    dto/                  # request / response DTO

migrations/               # SQL migrations
docs/                     # Swagger/OpenAPI documentation

docker-compose.yml
Dockerfile
```

### Responsibilities

**HTTP layer**

Отвечает за:

- parsing path/query parameters;
- decoding JSON;
- authentication context;
- преобразование domain models в HTTP DTO;
- HTTP status codes.

**Service layer**

Содержит:

- бизнес-логику;
- проверку membership и ролей;
- валидацию;
- orchestration repository/cache/policy.

**Repository layer**

Отвечает за:

- работу с MySQL;
- SQL-запросы;
- транзакции;
- optimistic locking;
- агрегированные SQL-отчёты.

**Redis cache**

Redis является вспомогательным cache layer.

MySQL остаётся source of truth. При недоступности Redis чтение списка задач продолжает работать напрямую через MySQL.

---

## Logging

The service uses structured JSON logging via Go's standard `log/slog` package.

Each HTTP request log contains:

- `request_id`
- `method`
- `path`
- `status`
- `duration_ms`
- `user_id` when available

Sensitive values such as passwords, password hashes, JWT tokens and secrets are never logged.

---

## Rate limiting

HTTP requests are limited by middleware using the token bucket algorithm from `golang.org/x/time/rate`.

Configuration is read from environment variables and can be changed without recompiling the application:

```env
RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20
```

Defaults:

- `RATE_LIMIT_RPS=10`
- `RATE_LIMIT_BURST=20`

Client identity:

- authenticated requests are identified by `user_id` from request context where available;
- unauthenticated requests are identified by `RemoteAddr` IP address;
- `RemoteAddr` port is ignored;
- `X-Forwarded-For` is not trusted by default.

Middleware order:

- `/api/v1/register` and `/api/v1/login` use a public rate limiter before authentication, so the client key is IP-based;
- protected endpoints run JWT authentication first and rate limiting second, so the client key is `user:<id>`;
- `/health` and Swagger routes are not rate limited.

When the bucket is exhausted, the API returns:

```text
HTTP 429 Too Many Requests
Retry-After: 1
```

`Retry-After` is an approximate minimal retry hint.

The rate limiter keeps per-client state in memory, removes stale clients after 10 minutes, and runs a single cleanup goroutine that is stopped during application shutdown.

The rate limiter is in-memory and applies per application instance. In a horizontally scaled deployment each instance maintains its own limiter state. A distributed limiter backed by Redis would be required for a global cluster-wide limit.

If the application is deployed behind a reverse proxy, production setup must include explicit trusted-proxy configuration before proxy headers such as `X-Forwarded-For` can be used for client identity.

---

## Running the project

### Requirements

Для запуска через Docker необходимы:

- Docker
- Docker Compose

Для локального запуска приложения дополнительно:

- Go 1.26+
- `golang-migrate`
- `swag` — если требуется регенерация Swagger

---

## Environment variables

Приложение конфигурируется через environment variables.

Создайте `.env` в корне проекта:

```env
HTTP_PORT=8080

MYSQL_ROOT_PASSWORD=root
MYSQL_HOST=mysql
MYSQL_PORT=3306
MYSQL_DATABASE=task_management
MYSQL_USER=task_user
MYSQL_PASSWORD=task_password

REDIS_HOST=redis
REDIS_PORT=6379

JWT_SECRET=local-development-secret-change-me
JWT_TTL_MINUTES=60

RATE_LIMIT_RPS=10
RATE_LIMIT_BURST=20
```

---

## Start with Docker Compose

Основной способ запуска проекта:

```bash
docker compose up --build
```

Запуск в background:

```bash
docker compose up -d --build
```

Docker Compose автоматически:

1. запускает MySQL и Redis;
2. ждёт готовности MySQL;
3. применяет database migrations из `migrations/`;
4. запускает API только после успешного завершения migrations.

Проверить состояние контейнеров:

```bash
docker compose ps
```

Остановить:

```bash
docker compose down
```

Clean start для development reset:

```bash
docker compose down -v
docker compose up --build
```

`docker compose down -v` удаляет локальные Docker volumes, включая данные MySQL. Не используйте эту команду для обычного restart, если нужно сохранить локальные данные.

После запуска API по умолчанию доступен на:

```text
http://localhost:8080
```

---

## Local run

Если MySQL и Redis запущены через Docker, а Go-приложение запускается локально, Docker DNS names `mysql` и `redis` с host-машины недоступны.

В таком случае можно использовать:

```bash
set -a
source .env
set +a

MYSQL_HOST=127.0.0.1 \
REDIS_HOST=127.0.0.1 \
go run ./cmd/api
```

---

## Database migrations

Миграции находятся в:

```text
migrations/
```

Для управления схемой используется `golang-migrate`.

При запуске через Docker Compose migrations применяются автоматически отдельным `migrate` service до старта API.

Ручной запуск нужен только для development/debugging, запуска приложения вне Compose или явного управления migration version.

Например:

```bash
migrate \
  -path migrations \
  -database "<MYSQL_DSN>" \
  up
```

Откат последней миграции:

```bash
migrate \
  -path migrations \
  -database "<MYSQL_DSN>" \
  down 1
```

Если в проекте используются Makefile targets, эти команды можно запускать через соответствующие `make`-команды.

---

## API documentation

Swagger UI доступен по адресу:

```text
http://localhost:8080/swagger/index.html
```

OpenAPI document:

```text
http://localhost:8080/swagger/doc.json
```

После изменения Swagger annotations документацию можно перегенерировать:

```bash
swag init \
  -g cmd/api/main.go \
  -o docs \
  --parseInternal
```

---

## API

### Health

| Method | Endpoint | Description |
|---|---|---|
| GET | `/health` | Health check |

### Authentication

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/register` | Register new user |
| POST | `/api/v1/login` | Login and receive JWT |

### Teams

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/teams` | Create team |
| GET | `/api/v1/teams` | List teams available to current user |
| POST | `/api/v1/teams/{id}/invite` | Add user to team |
| PATCH | `/api/v1/teams/{team_id}/members/{user_id}` | Update team member role (owner only) |

### Tasks

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/tasks` | Create task |
| GET | `/api/v1/tasks` | List tasks |
| GET | `/api/v1/tasks/{id}` | Get task |
| PUT | `/api/v1/tasks/{id}` | Update task |

### Task history

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/tasks/{id}/history` | Get task change history |

### Comments

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/tasks/{id}/comments` | Add task comment |
| GET | `/api/v1/tasks/{id}/comments` | List task comments |

### Statistics

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/teams/{team_id}/stats` | Get aggregated team statistics |

Полные request/response models доступны через Swagger UI.

---

## Authentication

Все защищённые endpoints требуют JWT:

```text
Authorization: Bearer <token>
```

Пример login:

```bash
curl -X POST \
  http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password"
  }'
```

Полученный token необходимо передавать в последующих запросах:

```bash
curl \
  http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer <token>"
```

---

## Team roles

Поддерживаются три роли:

```text
owner
admin
member
```

### Owner

Может:

- управлять командой;
- приглашать участников;
- менять роли участников между `member` и `admin`;
- редактировать любые задачи;
- просматривать статистику команды.

Передача ownership этим endpoint не поддерживается.

### Admin

Может:

- приглашать участников;
- редактировать любые задачи;
- просматривать статистику команды.

Не может менять роли участников.

### Member

Права зависят от отношения пользователя к задаче.

Не может менять роли участников.

Если `member` является создателем задачи, он может изменять:

- title;
- description;
- status;
- assignee.

Если `member` является только assignee, он может изменять только:

- status.

Обычный участник команды, который не является ни creator, ни assignee, может просматривать задачу, но не редактировать её.

Все участники команды могут:

- просматривать задачи;
- просматривать историю;
- читать комментарии;
- добавлять комментарии.

---

## Task filtering and pagination

Список задач:

```text
GET /api/v1/tasks
```

Поддерживает query parameters:

```text
team_id
status
assignee_id
limit
offset
```

Пример:

```bash
curl \
  "http://localhost:8080/api/v1/tasks?team_id=1&status=todo&assignee_id=2&limit=20&offset=0" \
  -H "Authorization: Bearer <token>"
```

`team_id` обязателен.

---

## Optimistic locking

Обновление задач защищено от lost update через optimistic locking.

Каждая задача содержит:

```json
{
  "version": 3
}
```

При обновлении клиент обязан отправить текущую версию задачи:

```json
{
  "title": "New title",
  "version": 3
}
```

На уровне SQL обновление выполняется по принципу:

```sql
UPDATE tasks
SET
    title = ?,
    description = ?,
    status = ?,
    assignee_id = ?,
    closed_at = ?,
    updated_at = NOW(),
    version = version + 1
WHERE id = ?
  AND version = ?;
```

Если другой запрос уже успел изменить задачу, старая версия больше не совпадает.

В этом случае API возвращает:

```text
409 Conflict
```

Это предотвращает ситуацию, когда два конкурентных обновления незаметно перезаписывают друг друга.

---

## Partial update semantics

По требованиям API используется:

```text
PUT /api/v1/tasks/{id}
```

При этом endpoint работает как частичное обновление:

- переданные поля изменяются;
- непереданные поля остаются без изменений.

Таким образом endpoint по семантике близок к `PATCH`, но HTTP method `PUT` сохранён в соответствии с API contract.

---

## Task history

Каждое успешное изменение задачи сохраняется в:

```text
task_history
```

История содержит только реально изменившиеся поля.

Например:

```json
{
  "title": {
    "old": "Implement API",
    "new": "Implement Task API"
  },
  "status": {
    "old": "todo",
    "new": "in_progress"
  }
}
```

Изменение задачи и создание history record выполняются в одной MySQL-транзакции:

```text
UPDATE task
      +
INSERT task_history
```

Если update не может быть выполнен, history record также не создаётся.

---

## Task status

Поддерживаемые статусы:

```text
todo
in_progress
done
cancelled
```

При переходе из открытого состояния в:

```text
done
cancelled
```

устанавливается `closed_at`.

При возврате задачи обратно в открытое состояние `closed_at` очищается.

---

## Redis caching

Список задач:

```text
GET /api/v1/tasks
```

кэшируется в Redis.

TTL cached result:

```text
5 minutes
```

Cache key зависит от:

- team ID;
- cache version;
- status;
- assignee;
- limit;
- offset.

Пример:

```text
tasks:1:v3:status=todo:assignee=2:limit=20:offset=0
```

### Cache invalidation

Для каждой команды используется отдельный version counter:

```text
tasks:{team_id}:cache_version
```

При создании или изменении задачи выполняется:

```text
INCR tasks:{team_id}:cache_version
```

Например:

```text
v3
 ↓
task updated
 ↓
v4
```

После этого приложение перестаёт читать старые `v3` cache keys и начинает использовать `v4`.

Старые ключи не удаляются через `KEYS` или `SCAN` — они автоматически исчезают после TTL.

Это позволяет инвалидировать весь task-list cache команды операцией `O(1)`.

### Redis failure

Redis не является source of truth.

Если cache недоступен:

```text
Redis error
    ↓
MySQL
    ↓
API response
```

Таким образом проблемы Redis не должны делать `GET /tasks` недоступным.

---

## Comments

Любой участник команды может добавить комментарий к задаче.

Перед сохранением content нормализуется через trimming whitespace.

Пустые комментарии не допускаются.

Пример:

```bash
curl -X POST \
  http://localhost:8080/api/v1/tasks/1/comments \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Looks good"
  }'
```

---

## Team statistics

Endpoint:

```text
GET /api/v1/teams/{team_id}/stats
```

доступен только:

```text
owner
admin
```

Отчёт включает:

- количество задач по статусам;
- top 3 assignees по количеству выполненных задач за последние 30 дней;
- среднее время закрытия задачи;
- общее количество комментариев по задачам команды.

Пример ответа:

```json
{
  "by_status": [
    {
      "status": "todo",
      "count": 4
    },
    {
      "status": "done",
      "count": 2
    }
  ],
  "top_assignees": [
    {
      "assignee_id": 2,
      "closed_count": 2
    }
  ],
  "avg_close_time_seconds": 4200.5,
  "total_comments": 8
}
```

Отчёт выполняется одним SQL-запросом с CTE.

Используются:

- `JOIN`;
- `GROUP BY`;
- `COUNT`;
- `AVG`;
- `TIMESTAMPDIFF`;
- date filtering;
- JSON aggregation.

N+1 queries для построения отчёта не используются.

---

## Error handling

Основные HTTP status codes:

| Status | Meaning |
|---|---|
| `200 OK` | Successful request |
| `201 Created` | Resource created |
| `400 Bad Request` | Invalid request or validation error |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | User does not have permission |
| `404 Not Found` | Resource not found |
| `409 Conflict` | Optimistic locking/version conflict |
| `500 Internal Server Error` | Unexpected infrastructure/internal error |

---

## Testing

Unit tests не требуют Docker:

```bash
go test ./internal/policy ./internal/service
```

MySQL repository integration tests 

```bash
go test -v ./internal/repository/mysql
```

Запуск всех Go tests:

```bash
go test ./...
```

Для расширенного вывода:

```bash
go test -v ./...
```

`go test ./...` включает Testcontainers-based MySQL integration test, поэтому для полного suite также требуется Docker.

Особенно важные сценарии для покрытия тестами:

- role-based task editing policy;
- optimistic locking;
- task validation;
- team permissions;
- task history;
- Redis cache behaviour;
- SQL statistics report.

Интеграционный тест статистического отчёта должен выполняться на реальном MySQL, поскольку запрос использует MySQL-specific SQL functionality.

---

## Swagger generation

Установка CLI:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Генерация документации:

```bash
swag init \
  -g cmd/api/main.go \
  -o docs \
  --parseInternal
```

После этого Swagger UI:

```text
http://localhost:8080/swagger/index.html
```

---

## Design decisions

### Raw SQL instead of ORM

Для работы с MySQL используется `database/sql` + `sqlx`.

Это позволяет:

- явно контролировать SQL;
- использовать transactions;
- реализовать optimistic locking;
- писать сложные аналитические запросы без ограничений ORM.

Особенно это важно для team statistics report, который использует CTE, aggregates, `JOIN` и date functions.

### Consumer-owned interfaces

Interfaces объявляются в слоях, которые их используют.

Например service определяет минимальный интерфейс repository/cache, необходимый конкретному use case.

Это:

- уменьшает coupling;
- упрощает mocking;
- облегчает unit testing.

### MySQL as source of truth

Redis используется только для ускорения чтения.

Данные в Redis всегда могут быть восстановлены из MySQL.

### Optimistic locking

Task versioning предотвращает lost updates без pessimistic database locks.

### Transactional history

Task update и audit history сохраняются атомарно в одной transaction.

### Versioned cache invalidation

Вместо поиска и удаления Redis keys используется increment cache version.

Это позволяет инвалидировать task-list cache команды без `KEYS`/`SCAN`.

---

## Security notes

- Passwords хранятся только в виде bcrypt hashes.
- Protected endpoints требуют JWT.
- Пользователь не может работать с задачами команды, членом которой он не является.
- Изменение задач дополнительно ограничивается ролью пользователя и его отношением к задаче.
- Team statistics доступны только `owner/admin`.
- Secrets и credentials передаются через environment variables и не должны храниться в Git.

---

## License

This project was created as a technical assignment.
