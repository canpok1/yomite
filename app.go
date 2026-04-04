//go:build gui

package main

import "context"

// NOTE: ctx をフィールドに保持するのは Wails v2 の標準パターン。
// OnStartup コールバックで渡され、Wails ランタイム API の呼び出しに必要。
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Greet(name string) string {
	return "Hello " + name + ", welcome to yomite!"
}
