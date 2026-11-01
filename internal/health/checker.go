package health

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Checker struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

type DependencyStatus struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ReadinessResult struct {
	Status       string                      `json:"status"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
}

func (c *Checker) Check(
	ctx context.Context,
) ReadinessResult {
	result := ReadinessResult{
		Status:       "ok",
		Dependencies: make(map[string]DependencyStatus),
	}

	if c.DB == nil {
		result.Dependencies["postgres"] = DependencyStatus{
			Status: "error",
			Error:  "database not configured",
		}
		result.Status = "degraded"
	} else {
		start := time.Now()
		err := c.DB.Ping(ctx)

		result.Dependencies["postgres"] = dependencyResult(
			start,
			err,
		)

		if err != nil {
			result.Status = "degraded"
		}
	}

	if c.Redis == nil {
		result.Dependencies["redis"] = DependencyStatus{
			Status: "error",
			Error:  "redis not configured",
		}
		result.Status = "degraded"
	} else {
		start := time.Now()
		err := c.Redis.Ping(ctx).Err()

		result.Dependencies["redis"] = dependencyResult(
			start,
			err,
		)

		if err != nil {
			result.Status = "degraded"
		}
	}

	return result
}

func dependencyResult(
	start time.Time,
	err error,
) DependencyStatus {
	result := DependencyStatus{
		Status:  "ok",
		Latency: time.Since(start).String(),
	}

	if err != nil {
		result.Status = "error"
		result.Error = sanitizeError(err)
	}

	return result
}

func sanitizeError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "dependency timeout"
	}

	return "dependency unavailable"
}
