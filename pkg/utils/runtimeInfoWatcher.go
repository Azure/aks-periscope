package utils

import (
	"time"
)

type RuntimeInfoWatcher struct {
	pollInterval time.Duration
	ticker       *time.Ticker
	err          error
	runtimeInfo  *RuntimeInfo
	handlers     []chan *RuntimeInfo
}

func NewRuntimeInfoWatcher(pollInterval time.Duration) *RuntimeInfoWatcher {
	runtimeInfo, err := GetRuntimeInfo()
	return &RuntimeInfoWatcher{
		pollInterval: pollInterval,
		ticker:       nil,
		err:          err,
		runtimeInfo:  runtimeInfo,
		handlers:     []chan *RuntimeInfo{},
	}
}

func (w *RuntimeInfoWatcher) AddHandler(ch chan *RuntimeInfo) {
	w.handlers = append(w.handlers, ch)
}

func (w *RuntimeInfoWatcher) handleUpdated(runtimeInfo *RuntimeInfo) {
	for _, handler := range w.handlers {
		go func(handler chan *RuntimeInfo) {
			handler <- runtimeInfo
		}(handler)
	}
}

func (w *RuntimeInfoWatcher) Start() {
	if w.ticker == nil {
		w.ticker = time.NewTicker(w.pollInterval)
		//w.errChan = make(chan error)
		//w.done = make(chan bool)
		go func() {
			for {
				select {
				// case <-done:
				// 	return
				case <-w.ticker.C:
					runtimeInfo, err := GetRuntimeInfo()
					if err != nil {
						w.err = err
					} else {
						w.handleUpdated(runtimeInfo)
					}
				}
			}
		}()
	}
}

func (w *RuntimeInfoWatcher) Get() (*RuntimeInfo, error) {
	return w.runtimeInfo, w.err
}

// func (w *ConfigWatcher) Stop() {
// 	if w.ticker != nil {
// 		w.ticker.Stop()
// 		w.done <- true
// 		w.ticker = nil
// 	}
// }

// func (w *ConfigWatcher) checkRunId() {
// 	w.fileSystem.FileExists(w.knownPaths.)
// 	err := wait.Poll(w.pollInterval, 0, func() (bool, error) {
// 		return collector.fileSystem.FileExists(completionNotificationPath)
// 	})
// }
