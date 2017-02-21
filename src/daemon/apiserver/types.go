package apiserver

import (
	"context"
	"daemon/payloads"
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

// Provider is an interface for task loaders
type Provider interface {
	LoadTasks(ctx context.Context) ([]Task, error)
}

// Task keeps exported task fields and instances
type Task struct {
	ID          string
	Image       string
	CPU         float32
	Mem         uint
	Constraints map[string]string
	Instances   []TaskInstance
	UpdatedAt   time.Time
}

// TaskInstance keeps exported instance fields
type TaskInstance struct {
	ID        string
	Addr      net.IP
	State     string
	Healthy   bool
	Payload   payloads.Payload
	UpdatedAt time.Time
}
