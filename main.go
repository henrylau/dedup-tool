package main

import (
	"embed"
	"folder-similarity/app/dto"
	"folder-similarity/app/service"
	"log"
	"os"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	defaultServerHost = "localhost"
	defaultServerPort = 8080
)

func serverHost() string {
	if h := os.Getenv("WAILS_SERVER_HOST"); h != "" {
		return h
	}
	return defaultServerHost
}

func serverPort() int {
	if p := os.Getenv("WAILS_SERVER_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			return n
		}
	}
	return defaultServerPort
}

//go:embed all:frontend/dist
var assets embed.FS

var similarityService = &service.Similarity{}

func init() {
	application.RegisterEvent[string]("log")
	application.RegisterEvent[dto.PhaseEvent]("phase")
	application.RegisterEvent[dto.ProgressEvent]("progress")
}

func main() {
	app := application.New(application.Options{
		Name:        "folder-similarity",
		Description: "Folder similarity and deduplication",
		Services: []application.Service{
			application.NewService(similarityService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Server: application.ServerOptions{
			Host: serverHost(),
			Port: serverPort(),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "folder-similarity",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	printStartupBanner()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
