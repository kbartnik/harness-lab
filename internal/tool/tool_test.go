package tool

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInSandbox(t *testing.T) {
	root := t.TempDir()

	cases := []struct {
		name     string
		input    string
		wantErr  bool
		wantKind ErrorKind
	}{
		{"simple relative", "foo.txt", false, ""},
		{"nested relative", "sub/foo.txt", false, ""},
		{"traversal", "../../etc/passwd", true, KindSandboxViolation},
		{"absolute", "/etc/passwd", true, KindSandboxViolation},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := resolveInSandbox(root, c.input)

			assert.Equal(t, c.wantErr, err != nil)

			if c.wantErr {
				var toolErr *ToolError
				require.True(t, errors.As(err, &toolErr))
				assert.Equal(t, c.wantKind, toolErr.Kind)
			}
		})
	}
}

func TestRead_Execute(t *testing.T) {
	t.Run("sandbox violation", func(t *testing.T) {
		root := t.TempDir()
		r := Read{Root: root}

		result, err := r.Execute(map[string]any{"path": "../../etc/passwd"})
		require.Error(t, err)

		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "sandbox_violation:"))
	})

	t.Run("not found", func(t *testing.T) {
		root := t.TempDir()
		r := Read{Root: root}

		result, err := r.Execute(map[string]any{"path": "does-not-exist.txt"})
		require.Error(t, err)

		var toolErr *ToolError
		require.True(t, errors.As(err, &toolErr))

		assert.Equal(t, KindNotFound, toolErr.Kind)
		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "not_found:"))
	})

	t.Run("empty file", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "empty.txt"), []byte{}, 0o644))

		r := Read{Root: root}
		result, err := r.Execute(map[string]any{"path": "empty.txt"})
		require.NoError(t, err)

		assert.False(t, result.IsError)
		assert.Equal(t, "", result.Output)
	})
}

func TestWrite_Execute(t *testing.T) {
	t.Run("writes file to sandbox", func(t *testing.T) {
		root := t.TempDir()
		w := Write{Root: root}

		result, err := w.Execute(map[string]any{"path": "foo.txt", "content": "hello"})
		require.NoError(t, err)

		assert.False(t, result.IsError)

		written, err := os.ReadFile(filepath.Join(root, "foo.txt"))
		require.NoError(t, err)

		assert.Equal(t, "hello", string(written))
	})

	t.Run("sandbox violation", func(t *testing.T) {
		root := t.TempDir()
		w := Write{Root: root}

		result, err := w.Execute(map[string]any{"path": "../../etc/passwd", "content": "malicious"})
		require.Error(t, err)

		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "sandbox_violation:"))
	})

	t.Run("permission denied", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "unreachable.txt"), []byte{}, 0o644))
		require.NoError(t, os.Chmod(filepath.Join(root, "unreachable.txt"), 0o444))
		w := Write{Root: root}

		result, err := w.Execute(map[string]any{"path": "unreachable.txt", "content": "hello"})
		require.Error(t, err)

		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "permission_denied:"))
	})
}
