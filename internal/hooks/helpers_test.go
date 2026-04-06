package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeScript writes content to a temporary executable shell script and returns
// its path. The file is cleaned up when the test finishes.
func writeScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hook.sh")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
	return path
}

// fileExists reports whether path exists on the filesystem.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
