package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStartFileWatcher_NoOp_WhenNoHooks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := NewManager(map[HookType][]HookConfig{})
	// Should not panic or error — just a no-op.
	StartFileWatcher(ctx, m, dir)
}

func TestStartFileWatcher_FiresOnWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	fired := make(chan string, 1)
	script := writeScript(t, `#!/bin/sh
payload=$(cat)
filename=$(echo "$payload" | jq -r '.data.filename')
echo "$filename" > `+filepath.Join(dir, "result.txt")+`
`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(map[HookType][]HookConfig{
		FileChanged: {{Command: script, Matcher: HookMatcher{Filename: ".env"}}},
	})
	StartFileWatcher(ctx, m, dir)

	// Give the watcher goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	// Write a matching file into the watched directory.
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("KEY=val"), 0o644))

	// Wait for the hook to fire and write the result.
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(filepath.Join(dir, "result.txt"))
		if err != nil {
			return false
		}
		fired <- string(data)
		return true
	}, 5*time.Second, 50*time.Millisecond)

	result := <-fired
	require.Contains(t, result, ".env")
}

func TestStartFileWatcher_SkipsNonMatchingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Hook only fires for ".envrc"; writing ".env" should not trigger it.
	sentinel := filepath.Join(dir, "fired.txt")
	script := writeScript(t, `#!/bin/sh
touch `+sentinel+`
`)
	m := NewManager(map[HookType][]HookConfig{
		FileChanged: {{Command: script, Matcher: HookMatcher{Filename: ".envrc"}}},
	})
	StartFileWatcher(ctx, m, dir)

	time.Sleep(50 * time.Millisecond)

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("x"), 0o644))

	// Sentinel must NOT appear within a reasonable window.
	time.Sleep(300 * time.Millisecond)
	require.False(t, fileExists(sentinel), "hook fired for non-matching file")
}

func TestStartFileWatcher_StopsOnContextCancel(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())

	m := NewManager(map[HookType][]HookConfig{
		FileChanged: {{Command: "true"}},
	})
	StartFileWatcher(ctx, m, dir)

	// Cancel and then write — should not panic.
	cancel()
	time.Sleep(50 * time.Millisecond)
	_ = os.WriteFile(filepath.Join(dir, "after-cancel.txt"), []byte("x"), 0o644)
	time.Sleep(100 * time.Millisecond)
}
