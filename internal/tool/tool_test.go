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

func newRead(t *testing.T) (Read, string) {
	t.Helper()
	root := t.TempDir()
	return Read{Root: root}, root
}

func newWrite(t *testing.T) (Write, string) {
	t.Helper()
	root := t.TempDir()
	return Write{Root: root}, root
}

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
		r, _ := newRead(t)

		result, err := r.Execute(map[string]any{"path": "../../etc/passwd"})
		require.NoError(t, err)

		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "sandbox_violation:"))
	})

	t.Run("not found", func(t *testing.T) {
		r, _ := newRead(t)

		result, err := r.Execute(map[string]any{"path": "does-not-exist.txt"})
		require.NoError(t, err)

		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "not_found:"))
	})

	t.Run("empty file", func(t *testing.T) {
		r, root := newRead(t)

		// create a zero-byte file at root
		require.NoError(t, os.WriteFile(filepath.Join(root, "empty.txt"), []byte{}, 0o644))

		result, err := r.Execute(map[string]any{"path": "empty.txt"})
		require.NoError(t, err)

		assert.False(t, result.IsError)
		assert.Equal(t, "", result.Output)
	})

	t.Run("permission denied", func(t *testing.T) {
		r, root := newRead(t)

		// create a zero byte file with the read bit unset
		require.NoError(t, os.WriteFile(filepath.Join(root, "noaccess.txt"), []byte{}, 0o200))
		require.NoError(t, os.Chmod(filepath.Join(root, "noaccess.txt"), 0o200))

		result, err := r.Execute(map[string]any{"path": "noaccess.txt"})
		require.NoError(t, err)

		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "permission_denied:"))
	})

	t.Run("missing path argument", func(t *testing.T) {
		r, _ := newRead(t)

		_, err := r.Execute(map[string]any{})
		require.Error(t, err)

		var toolErr *ToolError
		require.True(t, errors.As(err, &toolErr))

		assert.Equal(t, KindInvalidArgument, toolErr.Kind)
	})
}

func TestWrite_Execute(t *testing.T) {
	t.Run("writes file to sandbox", func(t *testing.T) {
		w, root := newWrite(t)

		result, err := w.Execute(map[string]any{"path": "foo.txt", "content": "hello"})
		require.NoError(t, err)

		assert.False(t, result.IsError)

		written, err := os.ReadFile(filepath.Join(root, "foo.txt"))
		require.NoError(t, err)

		assert.Equal(t, "hello", string(written))
	})

	t.Run("sandbox violation", func(t *testing.T) {
		w, _ := newWrite(t)

		result, err := w.Execute(map[string]any{"path": "../../etc/passwd", "content": "malicious"})
		require.NoError(t, err)

		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "sandbox_violation:"))
	})

	t.Run("permission denied", func(t *testing.T) {
		w, root := newWrite(t)

		// create a zero-byte file with the read bit unset
		require.NoError(t, os.WriteFile(filepath.Join(root, "unreachable.txt"), []byte{}, 0o644))
		require.NoError(t, os.Chmod(filepath.Join(root, "unreachable.txt"), 0o444))

		result, err := w.Execute(map[string]any{"path": "unreachable.txt", "content": "hello"})
		require.NoError(t, err)

		assert.True(t, result.IsError)
		assert.True(t, strings.HasPrefix(result.Output, "permission_denied:"))
	})

	t.Run("missing path argument", func(t *testing.T) {
		w, _ := newRead(t)

		_, err := w.Execute(map[string]any{"content": "hello"})
		require.Error(t, err)

		var toolErr *ToolError
		require.True(t, errors.As(err, &toolErr))
		assert.Equal(t, KindInvalidArgument, toolErr.Kind)
	})

	t.Run("missing content argument", func(t *testing.T) {
		w, _ := newWrite(t)

		_, err := w.Execute(map[string]any{"path": "foo.txt"})
		require.Error(t, err)

		var toolErr *ToolError
		require.True(t, errors.As(err, &toolErr))
		assert.Equal(t, KindInvalidArgument, toolErr.Kind)
	})
}

func TestWrite_Description(t *testing.T) {
	w, _ := newWrite(t)

	result := w.Description()

	assert.NotEmpty(t, result)
}

func TestRead_Description(t *testing.T) {
	r, _ := newRead(t)

	result := r.Description()

	assert.NotEmpty(t, result)
}
