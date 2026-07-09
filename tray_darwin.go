package main

/*
#cgo darwin LDFLAGS: -framework Cocoa
extern void tunlrDispatchStartTray(void);
extern void tunlrInstallReopenHandler(void);
*/
import "C"

// startTrayLoop runs the systray native loop on the macOS main thread (required
// for NSStatusItem) and installs the Dock-reopen handler.
func startTrayLoop() {
	C.tunlrDispatchStartTray()
	C.tunlrInstallReopenHandler()
}

//export tunlrStartTray
func tunlrStartTray() {
	if trayStart != nil {
		trayStart()
	}
}

//export tunlrReopen
func tunlrReopen() {
	if trayApp != nil {
		trayApp.showWindow()
	}
}
