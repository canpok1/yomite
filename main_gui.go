//go:build gui

package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/canpok1/yomite/internal/gui"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := gui.NewApp()

	err := wails.Run(&options.App{
		Title:  "yomite",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.Startup,
		Bind: []any{
			app,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %s\n", err.Error())
		os.Exit(1)
	}
}
