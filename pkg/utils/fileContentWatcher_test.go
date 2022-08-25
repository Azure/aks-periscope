package utils

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/aks-periscope/pkg/test"
)

func TestNewFileContentWatcher(t *testing.T) {
	watcher := NewFileContentWatcher(nil, 0)
	if watcher == nil {
		t.Errorf("expected FileSystemWatcher to be created")
	}
}

type fileEventNotification struct {
	content string
	isError bool
}

// file content change notification handler for testing purposes
type fileEventHandler struct {
	name                  string
	contentHandler        chan string
	errorHandler          chan error
	expectedNotifications []fileEventNotification
	finished              chan error
}

func newFileEventHandler(handlerName string) *fileEventHandler {
	return &fileEventHandler{
		name:                  handlerName,
		contentHandler:        make(chan string),
		errorHandler:          make(chan error),
		expectedNotifications: []fileEventNotification{},
		finished:              nil,
	}
}

func (h *fileEventHandler) expect(notifications []fileEventNotification) {
	h.expectedNotifications = notifications
	h.finished = make(chan error)

	go func() {
		for {
			// If there are no expected notifications then finish with no error.
			if len(h.expectedNotifications) == 0 {
				h.finished <- nil
			}

			var notification fileEventNotification
			select {
			case content := <-h.contentHandler:
				notification = fileEventNotification{content: content, isError: false}
			case <-h.errorHandler:
				notification = fileEventNotification{content: "", isError: true}
			}

			// If the notification was expected, remove it from the slice of expected notifications.
			wasExpected := false
			for i := range h.expectedNotifications {
				if reflect.DeepEqual(h.expectedNotifications[i], notification) {
					wasExpected = true
					h.expectedNotifications = append(h.expectedNotifications[:i], h.expectedNotifications[i+1:]...)
					break
				}
			}

			// Finish with an error if we encountered an unexpected notification.
			if !wasExpected {
				h.finished <- fmt.Errorf("unexpected notification for %s. Content: %s, Error: %v", h.name, notification.content, notification.isError)
			}
		}
	}()
}

// model of the test setup for one file content change handler
type fileEventHandlerSetup struct {
	name                    string
	filePaths               []string
	preChangeNotifications  []fileEventNotification
	postChangeNotifications []fileEventNotification
}

func TestStart(t *testing.T) {
	// For each test, we set up a fake filesystem with some files to be watched (and/or ignored).
	// We then watch those to check we get the appropriate notifications, then change the file contents
	// and check we get the right notifications.
	tests := []struct {
		name                  string
		unwatchedFileContents map[string]string
		initialFileContents   map[string]string
		changedFileContents   map[string]string
		errorPaths            []string
		handlerSetups         []fileEventHandlerSetup
	}{
		{
			name:                  "handler notified when file is initially read",
			unwatchedFileContents: map[string]string{},
			initialFileContents: map[string]string{
				"/testfile": "testfile_content",
			},
			changedFileContents: map[string]string{},
			errorPaths:          []string{},
			handlerSetups: []fileEventHandlerSetup{
				fileEventHandlerSetup{
					name:      "h1",
					filePaths: []string{"/testfile"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "testfile_content", isError: false},
					},
					postChangeNotifications: []fileEventNotification{},
				},
			},
		},
		{
			name: "handler not notified for unwatched file",
			unwatchedFileContents: map[string]string{
				"/unwatched": "unwatched_content",
			},
			initialFileContents: map[string]string{
				"/testfile": "testfile_content",
			},
			changedFileContents: map[string]string{},
			errorPaths:          []string{},
			handlerSetups: []fileEventHandlerSetup{
				fileEventHandlerSetup{
					name:      "h1",
					filePaths: []string{"/testfile"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "testfile_content", isError: false},
					},
					postChangeNotifications: []fileEventNotification{},
				},
			},
		},
		{
			name:                  "handler notified when file is changed",
			unwatchedFileContents: map[string]string{},
			initialFileContents: map[string]string{
				"/testfile": "testfile_content",
			},
			changedFileContents: map[string]string{
				"/testfile": "new_testfile_content",
			},
			errorPaths: []string{},
			handlerSetups: []fileEventHandlerSetup{
				fileEventHandlerSetup{
					name:      "h1",
					filePaths: []string{"/testfile"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "testfile_content", isError: false},
					},
					postChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "new_testfile_content", isError: false},
					},
				},
			},
		},
		{
			name:                  "handler notified when error reading file",
			unwatchedFileContents: map[string]string{},
			initialFileContents: map[string]string{
				"/testfile": "testfile_content",
			},
			changedFileContents: map[string]string{},
			errorPaths:          []string{"/testfile"},
			handlerSetups: []fileEventHandlerSetup{
				fileEventHandlerSetup{
					name:      "h1",
					filePaths: []string{"/testfile"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "", isError: true},
					},
					postChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "", isError: true},
					},
				},
			},
		},
		{
			name:                  "one handler per file - each file change notifies the correct handler",
			unwatchedFileContents: map[string]string{},
			initialFileContents: map[string]string{
				"/testfile1": "testfile1_content",
				"/testfile2": "testfile2_content",
			},
			changedFileContents: map[string]string{
				"/testfile1": "new_testfile1_content",
				"/testfile2": "new_testfile2_content",
			},
			errorPaths: []string{},
			handlerSetups: []fileEventHandlerSetup{
				fileEventHandlerSetup{
					name:      "h1",
					filePaths: []string{"/testfile1"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "testfile1_content", isError: false},
					},
					postChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "new_testfile1_content", isError: false},
					},
				},
				fileEventHandlerSetup{
					name:      "h2",
					filePaths: []string{"/testfile2"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "testfile2_content", isError: false},
					},
					postChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "new_testfile2_content", isError: false},
					},
				},
			},
		},
		{
			name:                  "one handler for multiple files - all files notify handler",
			unwatchedFileContents: map[string]string{},
			initialFileContents: map[string]string{
				"/testfile1": "testfile1_content",
				"/testfile2": "testfile2_content",
			},
			changedFileContents: map[string]string{
				"/testfile1": "new_testfile1_content",
				"/testfile2": "new_testfile2_content",
			},
			errorPaths: []string{},
			handlerSetups: []fileEventHandlerSetup{
				fileEventHandlerSetup{
					name:      "h1",
					filePaths: []string{"/testfile1", "/testfile2"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "testfile1_content", isError: false},
						fileEventNotification{content: "testfile2_content", isError: false},
					},
					postChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "new_testfile1_content", isError: false},
						fileEventNotification{content: "new_testfile2_content", isError: false},
					},
				},
			},
		},
		{
			name:                  "muiltiple handlers for one file - change notifies all handlers",
			unwatchedFileContents: map[string]string{},
			initialFileContents: map[string]string{
				"/testfile": "testfile_content",
			},
			changedFileContents: map[string]string{
				"/testfile": "new_testfile_content",
			},
			errorPaths: []string{},
			handlerSetups: []fileEventHandlerSetup{
				fileEventHandlerSetup{
					name:      "h1",
					filePaths: []string{"/testfile"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "testfile_content", isError: false},
					},
					postChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "new_testfile_content", isError: false},
					},
				},
				fileEventHandlerSetup{
					name:      "h2",
					filePaths: []string{"/testfile"},
					preChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "testfile_content", isError: false},
					},
					postChangeNotifications: []fileEventNotification{
						fileEventNotification{content: "new_testfile_content", isError: false},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := test.NewFakeFileSystem(map[string]string{})

			for path, content := range tt.unwatchedFileContents {
				fs.AddOrUpdateFile(path, content)
			}

			for path, content := range tt.initialFileContents {
				fs.AddOrUpdateFile(path, content)
			}

			for _, path := range tt.errorPaths {
				fs.SetFileAccessError(path, fmt.Errorf("expected error accessing %s", path))
			}

			// Create watcher and register handlers for specified file paths
			watcher := NewFileContentWatcher(fs, time.Microsecond)

			handlers := []*fileEventHandler{}
			for _, handlerSetup := range tt.handlerSetups {
				handler := newFileEventHandler(handlerSetup.name)
				handlers = append(handlers, handler)
				for _, path := range handlerSetup.filePaths {
					watcher.AddHandler(path, handler.contentHandler, handler.errorHandler)
				}
			}

			// Run test
			watcher.Start()

			// Start reading pre-change notifications for all handlers
			for i := range tt.handlerSetups {
				handlers[i].expect(tt.handlerSetups[i].preChangeNotifications)
			}

			// Wait for handlers to complete
			for _, handler := range handlers {
				err := <-handler.finished
				if err != nil {
					t.Errorf("handler %s finished pre-change watch with error: %v", handler.name, err)
				}
			}

			// Start reading post-change notifications for all handlers
			for i := range tt.handlerSetups {
				handlers[i].expect(tt.handlerSetups[i].postChangeNotifications)
			}

			// Update file contents
			for path, content := range tt.changedFileContents {
				fs.AddOrUpdateFile(path, content)
			}

			// Wait for handlers to complete
			for _, handler := range handlers {
				err := <-handler.finished
				if err != nil {
					t.Errorf("handler %s finished post-change watch with error: %v", handler.name, err)
				}
			}
		})
	}
}
