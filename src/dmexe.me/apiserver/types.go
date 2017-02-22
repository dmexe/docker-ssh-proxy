package apiserver

import (
	"context"
	"dmexe.me/payloads"
	"dmexe.me/utils"
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
	ID          string `json:"id"`
	Image       string `json:"image"`
	CPU         float32 `json:"cpu"`
	Mem         uint `json:"mem"`
	Constraints map[string]string `json:"constraints"`
	Instances   []TaskInstance `json:"instances"`
	UpdatedAt   time.Time `json:"updated_at"`
	Digest      string `json:"digest"`
}

// TaskInstance keeps exported instance fields
type TaskInstance struct {
	ID        string `json:"id"`
	Addr      net.IP `json:"addr"`
	State     string `json:"state"`
	Healthy   bool `json:"healthy"`
	Payload   payloads.Payload `json:"payload"`
	UpdatedAt time.Time `json:"updated_at"`
	Digest    string `json:"digest"`
}

// Result contains tasks and digest
type Result struct {
	Tasks     []Task `json:"tasks"`
	Digest    string `json:"digest"`
	CreatedAt time.Time `json:"created_at"`
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
