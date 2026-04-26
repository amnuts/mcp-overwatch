package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/wailsapp/wails/v3/pkg/application"

	"mcp-overwatch/internal/catalogue"
	"mcp-overwatch/internal/config"
	"mcp-overwatch/internal/database"
	"mcp-overwatch/internal/logging"
	"mcp-overwatch/internal/paths"
	"mcp-overwatch/internal/proxy"
	"mcp-overwatch/internal/runtime"
	"mcp-overwatch/internal/services"
	"mcp-overwatch/internal/stats"
	"mcp-overwatch/internal/supervisor"
)

// App holds all services and manages startup/shutdown lifecycle.
type App struct {
	Paths      *paths.Paths
	Config     *config.Config
	CatDB      *sql.DB
	StatsDB    *sql.DB
	Store      *catalogue.Store
	Registry   *catalogue.RegistryClient
	Runtimes   *runtime.Manager
	Supervisor *supervisor.Supervisor
	Proxy      *proxy.Proxy
	Logger     *logging.Logger
	Stats      *stats.Collector
	StatsStore *stats.Store

	CatalogueService *services.CatalogueService
	ServerService    *services.ServerService
	ImportService    *services.ImportService
	LogService       *services.LogService
	StatsService     *services.StatsService
	SettingsService  *services.SettingsService

	wailsApp *application.App
	systray  *application.SystemTray
}

// SetWailsApp sets the Wails application reference for event emission.
func (a *App) SetWailsApp(wailsApp *application.App) {
	a.wailsApp = wailsApp
	a.ImportService.SetWailsApp(wailsApp)
}

// SetSystemTray stores the system tray reference for dynamic tooltip updates.
func (a *App) SetSystemTray(systray *application.SystemTray) {
	a.systray = systray
}

// updateTrayTooltip refreshes the system tray tooltip with the current active server count.
func (a *App) updateTrayTooltip() {
	if a.systray == nil {
		return
	}
	count := len(a.Supervisor.ListActive())
	tooltip := fmt.Sprintf("MCP Overwatch \u2014 %d servers active", count)
	a.systray.SetTooltip(tooltip)
}

// emitEvent emits an event to the frontend if the Wails app is set.
func (a *App) emitEvent(name string, data ...any) {
	if a.wailsApp != nil {
		a.wailsApp.Event.Emit(name, data...)
	}
}

// NewApp wires everything together and returns a fully initialized App.
func NewApp() *App {
	a := &App{}

	var err error

	// 1. Resolve paths.
	base, err := paths.DefaultBase()
	if err != nil {
		log.Fatalf("resolve app data path: %v", err)
	}
	a.Paths = paths.New(base)

	// 2. Ensure all directories exist.
	if err := a.Paths.EnsureAll(); err != nil {
		log.Fatalf("create app directories: %v", err)
	}

	// 3. Load or create configuration.
	a.Config, err = config.LoadOrCreate(a.Paths.ConfigFile())
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 4. Open databases.
	a.CatDB, err = database.OpenCatalogue(a.Paths.CatalogueDB())
	if err != nil {
		log.Fatalf("open catalogue database: %v", err)
	}
	a.StatsDB, err = database.OpenStats(a.Paths.StatsDB())
	if err != nil {
		log.Fatalf("open stats database: %v", err)
	}

	// 5. Create stores.
	a.Store = catalogue.NewStore(a.CatDB)
	a.Registry = catalogue.NewRegistryClient(catalogue.DefaultRegistryURL)
	a.StatsStore = stats.NewStore(a.StatsDB)

	// 6. Create logger.
	logPath := filepath.Join(a.Paths.Logs(), "overwatch.log")
	a.Logger, err = logging.NewLogger(a.Config.Logging.RingBufferSize, logPath)
	if err != nil {
		log.Fatalf("create logger: %v", err)
	}
	// Wire logger callback to emit events to frontend.
	a.Logger.OnEntry(func(entry logging.Entry) {
		a.emitEvent("log:entry", entry)
	})

	// 7. Create stats collector.
	flushInterval := time.Duration(a.Config.Stats.FlushSeconds) * time.Second
	a.Stats = stats.NewCollector(a.StatsStore, flushInterval)

	// 8. Create runtime manager.
	a.Runtimes = runtime.NewManager(a.Paths.Runtimes())

	// 9. Create supervisor with callbacks.
	a.Supervisor = supervisor.New(
		func(serverID string, status supervisor.ServerStatus) {
			// Update store status.
			_ = a.Store.UpdateInstalledServerStatus(serverID, string(status))
			// Emit event to frontend (include is_active so the UI toggle stays in sync).
			evt := map[string]any{
				"id":     serverID,
				"status": string(status),
			}
			if srv, err := a.Store.GetInstalledServer(serverID); err == nil {
				evt["is_active"] = srv.IsActive
			}
			a.emitEvent("server:status", evt)
			// When a server starts, persist its discovered tools/resources/prompts to the DB.
			if status == supervisor.StatusRunning {
				if ms := a.Supervisor.Get(serverID); ms != nil {
					toolsJSON, _ := json.Marshal(ms.Tools())
					resourcesJSON, _ := json.Marshal(ms.Resources())
					promptsJSON, _ := json.Marshal(ms.Prompts())
					_ = a.Store.UpdateInstalledServerCachedData(serverID, string(toolsJSON), string(resourcesJSON), string(promptsJSON))
				}
			}
			// Rebuild proxy routes when a server starts or stops.
			if status == supervisor.StatusRunning || status == supervisor.StatusStopped || status == supervisor.StatusError {
				a.Proxy.RebuildRoutes()
			}
			// Update system tray tooltip with active server count.
			a.updateTrayTooltip()
		},
		func(serverID, stream, message string) {
			a.Logger.Add(logging.Entry{
				ServerID:  serverID,
				Direction: stream,
				Summary:   message,
			})
		},
	)

	// 10. Create proxy.
	a.Proxy = proxy.NewProxy(a.Config.Proxy.Port, &supervisorProvider{sup: a.Supervisor})
	a.Proxy.SetToolCallCallback(func(serverID, toolName string, latencyMs int64, eventType, errMsg string) {
		a.Stats.Record(stats.Event{
			ServerID:  serverID,
			EventType: eventType,
			ToolName:  toolName,
			LatencyMs: latencyMs,
			ErrorMsg:  errMsg,
		})
	})
	a.Proxy.SetLogCallback(func(serverID, stream, message string) {
		a.Logger.Add(logging.Entry{
			ServerID:  serverID,
			Direction: stream,
			Summary:   message,
		})
	})

	// 11. Wire services.
	a.CatalogueService = services.NewCatalogueService(a.Store, a.Registry, a.Logger, func(serverID, currentVersion, newVersion string) {
		a.emitEvent("server:updateAvailable", map[string]string{
			"id":             serverID,
			"currentVersion": currentVersion,
			"newVersion":     newVersion,
		})
	})
	a.ServerService = services.NewServerService(a.Store, a.Supervisor, a.Runtimes, a.Paths, a.Logger)
	a.ImportService = services.NewImportService(a.Store, a.Paths, a.Logger)
	a.LogService = services.NewLogService(a.Logger)
	a.StatsService = services.NewStatsService(a.StatsStore)
	a.SettingsService = services.NewSettingsService(a.Config, a.Paths.ConfigFile(), a.Paths, a.Runtimes, a.Store, a.Logger)

	return a
}

// OnStartup is called when the Wails app starts.
func (a *App) OnStartup() error {
	a.Logger.Add(logging.Entry{
		ServerID:  "system",
		Direction: "out",
		Summary:   "MCP Overwatch starting",
	})

	// 1. Reset all server statuses to stopped.
	if err := a.Store.ResetAllStatuses(); err != nil {
		log.Printf("warning: failed to reset server statuses: %v", err)
		a.Logger.Add(logging.Entry{
			ServerID:  "system",
			Direction: "in",
			Summary:   fmt.Sprintf("Failed to reset server statuses: %v", err),
		})
	}

	// 2. Start proxy in a goroutine (ListenAndServe blocks).
	go func() {
		if err := a.Proxy.Start(); err != nil {
			log.Printf("proxy stopped: %v", err)
		}
	}()

	// 3. Sync registry in background.
	go func() {
		a.Logger.Add(logging.Entry{
			ServerID:  "system",
			Direction: "out",
			Summary:   "Background registry sync started on launch",
		})
		result, err := a.Registry.SyncToStore(a.Store, func(serverID, currentVersion, newVersion string) {
			a.emitEvent("server:updateAvailable", map[string]string{
				"id":             serverID,
				"currentVersion": currentVersion,
				"newVersion":     newVersion,
			})
		})
		if err != nil {
			log.Printf("registry sync failed: %v", err)
			a.Logger.Add(logging.Entry{
				ServerID:  "system",
				Direction: "in",
				Summary:   fmt.Sprintf("Background registry sync failed: %v", err),
			})
		} else {
			a.Logger.Add(logging.Entry{
				ServerID:  "system",
				Direction: "in",
				Summary:   fmt.Sprintf("Background registry sync complete: %d new, %d updated", result.NewCount, result.UpdatedCount),
			})
		}
	}()

	// 4. Auto-start active servers.
	installed, err := a.Store.ListInstalledServers()
	if err != nil {
		log.Printf("warning: failed to list installed servers: %v", err)
		a.Logger.Add(logging.Entry{
			ServerID:  "system",
			Direction: "in",
			Summary:   fmt.Sprintf("Failed to list installed servers for auto-start: %v", err),
		})
		return nil
	}
	autoStartCount := 0
	for _, srv := range installed {
		if srv.IsActive {
			autoStartCount++
			srv := srv // capture loop var
			go func() {
				if err := a.ServerService.Toggle(srv.ID, true); err != nil {
					log.Printf("auto-start %s failed: %v", srv.ID, err)
				}
			}()
		}
	}
	if autoStartCount > 0 {
		a.Logger.Add(logging.Entry{
			ServerID:  "system",
			Direction: "out",
			Summary:   fmt.Sprintf("Auto-starting %d previously active server(s)", autoStartCount),
		})
	}

	return nil
}

// OnShutdown is called when the Wails app is shutting down.
func (a *App) OnShutdown() {
	a.Logger.Add(logging.Entry{
		ServerID:  "system",
		Direction: "out",
		Summary:   "MCP Overwatch shutting down",
	})
	a.Supervisor.Shutdown()
	_ = a.Stats.Stop()
	_ = a.Proxy.Stop()
	_ = a.Logger.Close()
	a.CatDB.Close()
	a.StatsDB.Close()
}

// supervisorProvider adapts the Supervisor to the proxy.ServerProvider interface.
type supervisorProvider struct {
	sup *supervisor.Supervisor
}

func (sp *supervisorProvider) CallTool(ctx context.Context, serverID, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	ms := sp.sup.Get(serverID)
	if ms == nil {
		return nil, fmt.Errorf("server %s not found", serverID)
	}
	c := ms.Client()
	if c == nil {
		return nil, fmt.Errorf("server %s not running", serverID)
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = args
	return c.CallTool(ctx, req)
}

func (sp *supervisorProvider) ListToolsForServer(serverID string) []mcp.Tool {
	ms := sp.sup.Get(serverID)
	if ms == nil {
		return nil
	}
	return ms.Tools()
}

func (sp *supervisorProvider) ActiveServers() []proxy.ServerInfo {
	active := sp.sup.ListActive()
	infos := make([]proxy.ServerInfo, len(active))
	for i, ms := range active {
		infos[i] = proxy.ServerInfo{
			ID:          ms.ID,
			DisplayName: ms.DisplayName,
		}
	}
	return infos
}
