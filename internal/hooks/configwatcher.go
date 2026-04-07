package hooks

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// StartConfigWatcher watches each path in configPaths (source → absolute path)
// for writes and fires ConfigChange hooks asynchronously when they change.
// No-ops if no ConfigChange hooks are configured or configPaths is empty.
func StartConfigWatcher(ctx context.Context, manager *Manager, configPaths map[string]string) {
	if len(configPaths) == 0 {
		return
	}

	manager.mu.Lock()
	configs := manager.hooks[ConfigChange]
	manager.mu.Unlock()

	if len(configs) == 0 {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("ConfigChange: failed to create watcher", "error", err)
		return
	}

	// Map absolute path → source name for fast lookup on events.
	// Watch parent directories (more reliable than watching files directly).
	pathToSource := make(map[string]string, len(configPaths))
	watched := make(map[string]struct{})
	for source, path := range configPaths {
		abs, err := filepath.Abs(path)
		if err != nil {
			slog.Warn("ConfigChange: cannot resolve path", "source", source, "path", path, "error", err)
			continue
		}
		pathToSource[abs] = source
		dir := filepath.Dir(abs)
		if _, ok := watched[dir]; !ok {
			if err := watcher.Add(dir); err != nil {
				slog.Warn("ConfigChange: failed to watch directory", "dir", dir, "error", err)
				continue
			}
			watched[dir] = struct{}{}
		}
	}

	if len(pathToSource) == 0 {
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
				if !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Create) {
					continue
				}
				abs, _ := filepath.Abs(ev.Name)
				source, ok := pathToSource[abs]
				if !ok {
					continue
				}
				go func() {
					_, _ = manager.Execute(context.Background(), ConfigChange, HookEvent{
						HookEventName: ConfigChange,
						RawEventData: map[string]string{
							"source": source,
							"path":   abs,
						},
					})
				}()
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Warn("ConfigChange watcher error", "error", err)
			}
		}
	}()
}
