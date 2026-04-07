package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStartConfigWatcher_NoOp_WhenNoHooks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := NewManager(map[HookType][]HookConfig{})
	StartConfigWatcher(ctx, m, map[string]string{"global": filepath.Join(dir, "crush.json")})
}

func TestStartConfigWatcher_NoOp_WhenNoConfigPaths(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := NewManager(map[HookType][]HookConfig{
		ConfigChange: {{Command: "true"}},
	})
	StartConfigWatcher(ctx, m, nil)
}

func TestStartConfigWatcher_FiresOnWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "crush.json")
	resultPath := filepath.Join(dir, "result.txt")

	script := writeScript(t, `#!/bin/sh
payload=$(cat)
source=$(echo "$payload" | jq -r '.data.source')
echo "$source" > `+resultPath+`
`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(map[HookType][]HookConfig{
		ConfigChange: {{Command: script}},
	})
	StartConfigWatcher(ctx, m, map[string]string{"global": cfgPath})

	time.Sleep(50 * time.Millisecond)

	require.NoError(t, os.WriteFile(cfgPath, []byte(`{}`), 0o600))

	require.Eventually(t, func() bool {
		data, err := os.ReadFile(resultPath)
		return err == nil && len(data) > 0
	}, 5*time.Second, 50*time.Millisecond)

	data, _ := os.ReadFile(resultPath)
	require.Contains(t, string(data), "global")
}

func TestStartConfigWatcher_SkipsUnwatchedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "crush.json")
	sentinel := filepath.Join(dir, "fired.txt")

	script := writeScript(t, `#!/bin/sh
touch `+sentinel+`
`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(map[HookType][]HookConfig{
		ConfigChange: {{Command: script}},
	})
	// Watch crush.json but write a different file.
	StartConfigWatcher(ctx, m, map[string]string{"global": cfgPath})

	time.Sleep(50 * time.Millisecond)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.json"), []byte(`{}`), 0o600))

	time.Sleep(300 * time.Millisecond)
	require.False(t, fileExists(sentinel), "hook fired for unwatched file")
}

func TestStartConfigWatcher_PayloadHasSourceAndPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "crush.json")
	resultPath := filepath.Join(dir, "result.txt")

	script := writeScript(t, `#!/bin/sh
payload=$(cat)
name=$(echo "$payload" | jq -r '.hook_event_name')
[ "$name" = "ConfigChange" ] || { echo "wrong event: $name" >&2; exit 2; }
source=$(echo "$payload" | jq -r '.data.source')
path=$(echo "$payload" | jq -r '.data.path')
echo "$source:$path" > `+resultPath+`
`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(map[HookType][]HookConfig{
		ConfigChange: {{Command: script}},
	})
	StartConfigWatcher(ctx, m, map[string]string{"workspace": cfgPath})

	time.Sleep(50 * time.Millisecond)
	require.NoError(t, os.WriteFile(cfgPath, []byte(`{}`), 0o600))

	require.Eventually(t, func() bool {
		data, err := os.ReadFile(resultPath)
		return err == nil && len(data) > 0
	}, 5*time.Second, 50*time.Millisecond)

	data, _ := os.ReadFile(resultPath)
	require.Contains(t, string(data), "workspace:")
	require.Contains(t, string(data), cfgPath)
}
