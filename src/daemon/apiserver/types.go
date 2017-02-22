package apiserver

import (
	"context"
	"daemon/payloads"
	"daemon/utils"
	"net"
	"time"
)

const (
	// TaskStatusPending when is still not running
	TaskStatusPending = "pending"

	// TaskStatusRunning when is running
	TaskStatusRunning = "running"

	// TaskStatusFailed when failed for any reason
	TaskStatusFailed = "failed"

	// TaskStatusFinished when successfully finished
	TaskStatusFinished = "finished"

	// TaskStatusUnknown all other statuses
	TaskStatusUnknown = "unknown"
)

// Task keeps exported task fields and instances
type Task struct {
	ID          string
	Image       string
	CPU         float32
	Mem         uint
	Constraints map[string]string
	Instances   []TaskInstance
	UpdatedAt   time.Time
	Digest      string
}

// TaskInstance keeps exported instance fields
type TaskInstance struct {
	ID        string
	Addr      net.IP
	State     string
	Healthy   bool
	Payload   payloads.Payload
	UpdatedAt time.Time
	Digest    string
}

// Result contains tasks and digest
type Result struct {
	Tasks     []Task
	Digest    string
	CreatedAt time.Time
}

// Provider is an interface for task loaders
type Provider interface {
	GetTasks(ctx context.Context) (Result, error)
}

// RunnableProvider is an interface for aggregator
type RunnableProvider interface {
	Provider
	utils.Runnable
}
