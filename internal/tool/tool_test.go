package tool

import (
	"errors"
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
