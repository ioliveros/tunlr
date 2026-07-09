package main

import (
	"context"
	_ "embed"
	"runtime"

	"github.com/energye/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)


//go:embed build/appicon.png
var trayIconPNG []byte

//go:embed build/windows/icon.ico
var trayIconICO []byte

var trayStart func()

var trayApp *App

// initTray creates the tray icon/menu and starts the tray loop alongside Wails.
// The window hides to the tray on close; the menu restores it or quits.
func (a *App) initTray() {
	trayApp = a
	onReady := func() {
		if runtime.GOOS == "windows" {
			systray.SetIcon(trayIconICO)
		} else {
			systray.SetIcon(trayIconPNG)
		}
		systray.SetTooltip("tunlr")

		systray.SetOnClick(func(systray.IMenu) { a.showWindow() })

		show := systray.AddMenuItem("Show tunlr", "Show the window")
		show.Click(a.showWindow)
		systray.AddSeparator()
		quit := systray.AddMenuItem("Quit", "Quit tunlr")
		quit.Click(a.quit)
	}

	start, end := systray.RunWithExternalLoop(onReady, func() {})
	a.trayEnd = end
	trayStart = start
	startTrayLoop()
}

// showWindow restores and focuses the main window from the tray or Dock.
func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.WindowShow(a.ctx)
}

// quit exits for real (tray "Quit"), bypassing the hide-to-tray behaviour.
func (a *App) quit() {
	a.quitting = true
	wailsruntime.Quit(a.ctx)
}

// beforeClose intercepts the window close button: hide to the tray and keep
// tunnels running instead of quitting, unless we're quitting via the tray.
func (a *App) beforeClose(context.Context) (prevent bool) {
	if a.quitting {
		return false
	}
	wailsruntime.WindowHide(a.ctx)
	return true
}

// shutdown stops the tray loop when the app exits.
func (a *App) shutdown(context.Context) {
	if a.trayEnd != nil {
		a.trayEnd()
	}
}
