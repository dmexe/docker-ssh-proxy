package utils

// Runnable is simple interface for services
type Runnable interface {
	Run() error
	Wait() error
}
