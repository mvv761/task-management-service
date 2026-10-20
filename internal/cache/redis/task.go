package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"task-management-service/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

type TaskCache struct {
	client *goredis.Client
}

func NewTaskCache(
	client *goredis.Client,
) *TaskCache {
	return &TaskCache{
		client: client,
	}
}

func (c *TaskCache) Version(ctx context.Context, teamID uint64) (uint64, error) {
	key := fmt.Sprintf(
		"tasks:%d:cache_version",
		teamID,
	)

	value, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return 0, nil
		}

		return 0, fmt.Errorf(
			"get task cache version: %w",
			err,
		)
	}

	version, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"parse task cache version: %w",
			err,
		)
	}

	return version, nil
}

func (c *TaskCache) Invalidate(ctx context.Context, teamID uint64) error {
	key := fmt.Sprintf(
		"tasks:%d:cache_version",
		teamID,
	)

	if err := c.client.Incr(ctx, key).Err(); err != nil {
		return fmt.Errorf(
			"invalidate task cache: %w",
			err,
		)
	}

	return nil
}

func taskListKey(filter domain.TaskFilter, version uint64) string {
	status := "all"
	if filter.Status != nil {
		status = string(*filter.Status)
	}

	assignee := "all"
	if filter.AssigneeID != nil {
		assignee = strconv.FormatUint(
			*filter.AssigneeID,
			10,
		)
	}

	return fmt.Sprintf(
		"tasks:%d:v%d:status=%s:assignee=%s:limit=%d:offset=%d",
		filter.TeamID,
		version,
		status,
		assignee,
		filter.Limit,
		filter.Offset,
	)
}

func (c *TaskCache) Get(ctx context.Context, filter domain.TaskFilter, version uint64) ([]domain.Task, bool, error) {
	key := taskListKey(filter, version)

	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get cached tasks: %w", err)
	}

	var tasks []domain.Task

	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached tasks: %w", err)
	}

	return tasks, true, nil
}

func (c *TaskCache) Set(ctx context.Context, filter domain.TaskFilter, version uint64, tasks []domain.Task, ttl time.Duration) error {
	key := taskListKey(filter, version)

	data, err := json.Marshal(tasks)
	if err != nil {
		return fmt.Errorf("marshal tasks: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set cached tasks: %w", err)
	}

	return nil
}
