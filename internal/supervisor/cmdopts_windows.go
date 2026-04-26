//go:build windows

package supervisor

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/mark3labs/mcp-go/client/transport"
)

// platformCommandFunc returns a CommandFunc that sets CREATE_NO_WINDOW on Windows
// to prevent console windows from appearing for spawned MCP server processes.
func platformCommandFunc(workDir string) transport.CommandFunc {
	return func(ctx context.Context, command string, env []string, args []string) (*exec.Cmd, error) {
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Env = append(os.Environ(), env...)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
		if workDir != "" {
			cmd.Dir = workDir
		}
		return cmd, nil
	}
}
