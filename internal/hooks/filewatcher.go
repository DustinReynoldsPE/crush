package hooks

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// StartFileWatcher watches workDir for file writes/creates and fires FileChanged
// hooks for files whose basenames match configured Filename matchers. The goroutine
// stops when ctx is cancelled. No-ops if no FileChanged hooks are configured.
func StartFileWatcher(ctx context.Context, manager *Manager, workDir string) {
	manager.mu.Lock()
	configs := manager.hooks[FileChanged]
	manager.mu.Unlock()

	if len(configs) == 0 {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("FileChanged: failed to create watcher", "error", err)
		return
	}

	if err := watcher.Add(workDir); err != nil {
		slog.Error("FileChanged: failed to watch directory", "dir", workDir, "error", err)
		watcher.Close()
		return
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) {
					absPath, _ := filepath.Abs(ev.Name)
					basename := filepath.Base(absPath)
					go func() {
						_, _ = manager.Execute(context.Background(), FileChanged, HookEvent{
							HookEventName: FileChanged,
							RawEventData: map[string]string{
								"path":     absPath,
								"filename": basename,
							},
						})
					}()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Warn("FileChanged watcher error", "error", err)
			}
		}
	}()
}
