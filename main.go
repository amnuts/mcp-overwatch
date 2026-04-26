package main

import (
	"embed"
	_ "embed"
	"log"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	mcpapp "mcp-overwatch/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/tray-icon.png
var trayIconBytes []byte

func main() {
	// Wire up the application with all domain services.
	a := mcpapp.NewApp()

	// Create the Wails application.
	wailsApp := application.New(application.Options{
		Name:        "MCP Overwatch",
		Description: "A unified manager for MCP servers",
		// Suppress per-binding-call info logs (Wails 3 alpha logs every result
		// payload at info, which dumps multi-MB JSON to the dev terminal for
		// stats endpoints).
		LogLevel: slog.LevelWarn,
		Services: []application.Service{
			application.NewService(a.CatalogueService),
			application.NewService(a.ServerService),
			application.NewService(a.ImportService),
			application.NewService(a.LogService),
			application.NewService(a.StatsService),
			application.NewService(a.SettingsService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		OnShutdown: a.OnShutdown,
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// Give the app a reference to the Wails app for event emission.
	a.SetWailsApp(wailsApp)

	// Run startup logic (reset statuses, start proxy, sync registry, auto-start servers).
	if err := a.OnStartup(); err != nil {
		log.Fatalf("startup failed: %v", err)
	}

	// Create the main window with frameless style for custom title bar.
	mainWindow := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "MCP Overwatch",
		Width:     1200,
		Height:    800,
		Frameless: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(15, 20, 25),
		URL:              "/",
	})

	// Set up system tray.
	trayMenu := application.NewMenu()
	trayMenu.Add("Open").OnClick(func(ctx *application.Context) {
		mainWindow.Show().Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		wailsApp.Quit()
	})

	systray := wailsApp.SystemTray.New()
	systray.SetIcon(trayIconBytes)
	systray.SetTooltip("MCP Overwatch \u2014 0 servers active")
	systray.SetMenu(trayMenu)
	systray.OnClick(func() {
		mainWindow.Show().Focus()
	})

	// Store systray reference in the app for dynamic tooltip updates.
	a.SetSystemTray(systray)

	// Run the application (blocks until exit).
	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
