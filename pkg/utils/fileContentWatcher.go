package utils

import (
	"io"
	"log"
	"time"
)

type fileContentItem struct {
	content         string
	err             error
	contentHandlers []chan string
	errorHandlers   []chan error
}

type FileContentWatcher struct {
	fileSystem   *FileSystem
	pollInterval time.Duration
	ticker       *time.Ticker
	items        map[string]*fileContentItem
}

func NewFileContentWatcher(fileSystem *FileSystem, pollInterval time.Duration) *FileContentWatcher {
	return &FileContentWatcher{
		fileSystem:   fileSystem,
		pollInterval: pollInterval,
		ticker:       nil,
		items:        map[string]*fileContentItem{},
	}
}

func (w *FileContentWatcher) AddHandler(filePath string, contentChan chan string, errChan chan error) {
	if item, ok := w.items[filePath]; ok {
		w.items[filePath].contentHandlers = append(item.contentHandlers, contentChan)
		w.items[filePath].errorHandlers = append(item.errorHandlers, errChan)
	} else {
		w.items[filePath] = &fileContentItem{
			content:         "",
			err:             nil,
			contentHandlers: []chan string{contentChan},
			errorHandlers:   []chan error{errChan},
		}
	}
}

func (item *fileContentItem) handleUpdated(filePath string) {
	if item.err != nil {
		for _, handler := range item.errorHandlers {
			go func(handler chan error) {
				log.Printf("Sending error to handler channel for %s:\n%v", filePath, item.err)
				handler <- item.err
			}(handler)
		}
	} else {
		for _, handler := range item.contentHandlers {
			go func(handler chan string) {
				log.Printf("Sending content to handler channel for %s:\n%s", filePath, item.content)
				handler <- item.content
			}(handler)
		}
	}
}

func (w *FileContentWatcher) checkFilePaths() {
	for filePath, item := range w.items {
		content, err := GetContent(func() (io.ReadCloser, error) { return w.fileSystem.GetFileReader(filePath) })
		if err != nil {
			item.err = err
		} else if content != item.content {
			log.Printf("File %s updated", filePath)
			item.content = content
			item.handleUpdated(filePath)
		}
	}
}

func (w *FileContentWatcher) Start() {
	if w.ticker == nil {
		w.ticker = time.NewTicker(w.pollInterval)

		go func() {
			w.checkFilePaths()
			for {
				<-w.ticker.C
				w.checkFilePaths()
			}
		}()
	}
}
