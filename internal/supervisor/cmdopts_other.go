//go:build !windows

package supervisor

import (
	"context"
	"os"
	"os/exec"

	"github.com/mark3labs/mcp-go/client/transport"
)

// platformCommandFunc returns a CommandFunc that optionally sets WorkDir.
// On non-Windows platforms, no special process attributes are needed.
func platformCommandFunc(workDir string) transport.CommandFunc {
	if workDir == "" {
		return nil
	}
	return func(ctx context.Context, command string, env []string, args []string) (*exec.Cmd, error) {
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Env = append(os.Environ(), env...)
		cmd.Dir = workDir
		return cmd, nil
	}
}
