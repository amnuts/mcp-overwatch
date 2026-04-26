package supervisor

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

// transportKind returns the canonical transport identifier for ms, defaulting
// to "stdio" when the field is empty (preserves behavior for older installs
// and imported servers that never set the field).
func transportKind(ms *ManagedServer) string {
	t := strings.ToLower(strings.TrimSpace(ms.TransportType))
	switch t {
	case "":
		return "stdio"
	case "http", "streaming-http", "streamable_http":
		return "streamable-http"
	default:
		return t
	}
}

// validateTransport checks that the ManagedServer has the fields required for
// its transport type before any I/O is attempted.
func validateTransport(ms *ManagedServer) error {
	switch transportKind(ms) {
	case "stdio":
		if strings.TrimSpace(ms.Command) == "" {
			return fmt.Errorf("stdio transport requires a command — none configured (the server may need a runtime that is not installed)")
		}
		return nil
	case "sse", "streamable-http":
		if strings.TrimSpace(ms.RemoteURL) == "" {
			return fmt.Errorf("%s transport requires a remote URL — none configured", transportKind(ms))
		}
		return nil
	default:
		return fmt.Errorf("unsupported transport type: %q", ms.TransportType)
	}
}

// buildClient constructs an mcp-go client for the configured transport.
// Returns the client, a flag indicating whether the caller must invoke
// client.Start (true for remote transports; stdio auto-starts inside the
// constructor), and any error from constructing the transport.
func buildClient(ms *ManagedServer) (*client.Client, bool, error) {
	switch transportKind(ms) {
	case "stdio":
		var opts []transport.StdioOption
		if cmdFunc := platformCommandFunc(ms.WorkDir); cmdFunc != nil {
			opts = append(opts, transport.WithCommandFunc(cmdFunc))
		}
		c, err := client.NewStdioMCPClientWithOptions(ms.Command, ms.Env, ms.Args, opts...)
		if err != nil {
			return nil, false, err
		}
		return c, false, nil

	case "sse":
		var opts []transport.ClientOption
		if len(ms.Headers) > 0 {
			opts = append(opts, transport.WithHeaders(ms.Headers))
		}
		c, err := client.NewSSEMCPClient(ms.RemoteURL, opts...)
		if err != nil {
			return nil, false, err
		}
		return c, true, nil

	case "streamable-http":
		var opts []transport.StreamableHTTPCOption
		if len(ms.Headers) > 0 {
			opts = append(opts, transport.WithHTTPHeaders(ms.Headers))
		}
		c, err := client.NewStreamableHttpClient(ms.RemoteURL, opts...)
		if err != nil {
			return nil, false, err
		}
		return c, true, nil

	default:
		return nil, false, fmt.Errorf("unsupported transport type: %q", ms.TransportType)
	}
}
