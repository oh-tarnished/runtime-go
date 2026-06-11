package config

import (
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/oh-tarnished/runtime-go/config/shared"
)

// FileEvent represents a file change event with its content.
type FileEvent struct {
	// Path is the file path that changed.
	Path string
	// Content is the file content read immediately on change.
	Content []byte
	// Timestamp indicates when the event was processed.
	Timestamp time.Time
}

// WatcherOptions holds optional configuration values.
type WatcherOptions struct {
	// OverridePollInterval is provided for future configuration (currently unused).
	OverridePollInterval time.Duration
}

// Watcher monitors a file or directory and enqueues file content events.
type Watcher struct {
	// Path is the file or directory to watch.
	Path string
	// isDir indicates whether Path is a directory.
	isDir bool
	// WatcherOptions holds configuration options.
	WatcherOptions *WatcherOptions
	// QueueSize is the size of the internal file path queue.
	QueueSize int

	// Internal fsnotify watcher.
	watcher *fsnotify.Watcher

	// queue holds file paths that have changed.
	queue chan string
	// events is the public channel where FileEvents are sent.
	events chan FileEvent

	// done signals the goroutines to stop.
	done chan struct{}
}

// newWatcher creates a new generic Watcher for a file or directory.
// Returns an error if the given path does not exist.
func newWatcher(path string, options *WatcherOptions) (*Watcher, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	shared.Pulse.Logger.Debugf("Creating new watcher for path '%s' (isDir: %t)", path, info.IsDir())
	return &Watcher{
		Path:           path,
		isDir:          info.IsDir(),
		WatcherOptions: options,
		QueueSize:      25, // default queue size; adjust as needed.
		done:           make(chan struct{}),
	}, nil
}

// Start initializes the fsnotify watcher, sets up channels, and starts the internal loops.
func (w *Watcher) Start() error {
	var err error
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		shared.Pulse.Logger.Errorf("Watcher failed to create fsnotify instance: %v", err)
		return err
	}

	// Create the internal queue and events channel.
	w.queue = make(chan string, w.QueueSize)
	// Buffer the events channel if you expect bursts.
	w.events = make(chan FileEvent, w.QueueSize)

	// Add the target (file or directory) to the watcher.
	if err := w.watcher.Add(w.Path); err != nil {
		shared.Pulse.Logger.Errorf("Watcher failed to add path '%s': %v", w.Path, err)
		return err
	}

	// Launch the fsnotify event loop.
	go w.eventLoop()
	// Launch a worker to process file changes.
	go w.processQueue()

	shared.Pulse.Logger.Infof("Started watching %s: %s", w.targetType(), w.Path)
	return nil
}

// Stop stops the watcher and terminates the goroutines.
func (w *Watcher) Stop() {
	shared.Pulse.Logger.Debugf("Stopping watcher for path '%s'", w.Path)
	close(w.done)
	if w.watcher != nil {
		if err := w.watcher.Close(); err != nil {
			shared.Pulse.Logger.Errorf("Watcher failed to close fsnotify instance: %v", err)
		}
	}
}

// Events returns a read-only channel where FileEvents are sent.
func (w *Watcher) Events() <-chan FileEvent {
	return w.events
}

// targetType returns a string description of what is being watched.
func (w *Watcher) targetType() string {
	if w.isDir {
		return "directory"
	}
	return "file"
}

// eventLoop listens for fsnotify events and enqueues changed file paths.
func (w *Watcher) eventLoop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			shared.Pulse.Logger.Debugf("Watcher received fsnotify event: %s", event)
			// For a single file watcher, ensure we are processing the correct file.
			if !w.isDir && event.Name != w.Path {
				continue
			}
			// Process create and write events.
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				select {
				case w.queue <- event.Name:
					shared.Pulse.Logger.Debugf("Enqueued %s change: %s", w.targetType(), event.Name)
				default:
					shared.Pulse.Logger.Warnf("Queue full. Dropping event for %s: %s", w.targetType(), event.Name)
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			shared.Pulse.Logger.Errorf("Watcher error: %v", err)
		case <-w.done:
			return
		}
	}
}

// processQueue reads file paths from the internal queue, reads their contents,
// and sends a FileEvent on the public events channel.
func (w *Watcher) processQueue() {
	for {
		select {
		case file := <-w.queue:
			w.handleFileEvent(file)
		case <-w.done:
			return
		}
	}
}

// handleFileEvent reads the file content quickly and publishes a FileEvent.
// If the target is a directory, it only processes files.
func (w *Watcher) handleFileEvent(file string) {
	// Check if the file still exists and is not a directory.
	info, err := os.Stat(file)
	if err != nil {
		shared.Pulse.Logger.Errorf("Watcher failed to stat file '%s': %v", file, err)
		return
	}
	if info.IsDir() {
		shared.Pulse.Logger.Debugf("Watcher skipping directory change: %s", file)
		return
	}

	// Read the file contents immediately.
	content, err := os.ReadFile(file)
	if err != nil {
		shared.Pulse.Logger.Errorf("Watcher failed to read file '%s': %v", file, err)
		return
	}

	fe := FileEvent{
		Path:      file,
		Content:   content,
		Timestamp: time.Now(),
	}

	// Use blocking send so that the event is queued until a consumer processes it.
	w.events <- fe
	shared.Pulse.Logger.Debugf("Published FileEvent for: %s (%d bytes)", file, len(content))
}
