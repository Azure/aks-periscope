package utils

import (
	"log"
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
			log.Print("Sending runtime info to handler channel")
			handler <- runtimeInfo
		}(handler)
	}
}

func (w *RuntimeInfoWatcher) Start() {
	if w.ticker == nil {
		w.ticker = time.NewTicker(w.pollInterval)

		go func() {
			for {
				<-w.ticker.C
				log.Print("Getting runtime info")
				runtimeInfo, err := GetRuntimeInfo()
				if err != nil {
					w.err = err
				} else {
					log.Print("Handle runtime info update")
					w.handleUpdated(runtimeInfo)
				}
			}
		}()
	}
}

func (w *RuntimeInfoWatcher) Get() (*RuntimeInfo, error) {
	return w.runtimeInfo, w.err
}
