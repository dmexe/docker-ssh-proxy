package utils

import "sync"

// Runnable is simple interface for services
type Runnable interface {
	Run(wg *sync.WaitGroup) error
}
