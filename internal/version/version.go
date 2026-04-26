package version

// Version is the app version, overridden at build time via:
//
//	go build -ldflags "-X 'mcp-overwatch/internal/version.Version=1.2.3'"
var Version = "1.0.0-alpha"
