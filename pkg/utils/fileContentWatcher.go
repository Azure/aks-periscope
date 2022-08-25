package utils

import (
	"io"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
)

type fileContentItem struct {
	content         string
	err             error
	contentHandlers []chan string
	errorHandlers   []chan error
}

// FileContentWatcher allows clients to register to receive notifications via a channel when a file's content changes
// or there is an error reading that file. It uses polling and stores file content in memory, valuing simplicity over
// sophisticated approaches involving cross-platform inotify or hashing mechanisms. With that in mind, it is appropriate
// for watching a small number of small files.
type FileContentWatcher struct {
	fileSystem   interfaces.FileSystemAccessor
	pollInterval time.Duration
	ticker       *time.Ticker
	items        map[string]*fileContentItem
}

// NewFileContentWatcher constructs a FileContentWatcher based on the specified FileSystemAccessor and polling interval.
// This will initially contain no handlers, and will not start polling until the Start method is called.
func NewFileContentWatcher(fileSystem interfaces.FileSystemAccessor, pollInterval time.Duration) *FileContentWatcher {
	return &FileContentWatcher{
		fileSystem:   fileSystem,
		pollInterval: pollInterval,
		ticker:       nil,
		items:        map[string]*fileContentItem{},
	}
}

// AddHandler supplies channels for receiving notifications when the specified file is read or changed, or when there is
// an error reading it. No files will be read or notifications sent until the Start method is called.
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
				handler <- item.err
			}(handler)
		}
	} else {
		for _, handler := range item.contentHandlers {
			go func(handler chan string) {
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
			item.handleUpdated(filePath)
		} else if content != item.content {
			item.content = content
			item.handleUpdated(filePath)
		}
	}
}

// Start tells the FileContentWatcher to periodically read the files for which a handler has been registered,
// starting immediately.
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
