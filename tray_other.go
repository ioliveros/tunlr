//go:build !darwin

package main

// startTrayLoop runs the systray native loop directly. 
// Only macOS requires the loop to start on the main thread.
func startTrayLoop() {
	if trayStart != nil {
		trayStart()
	}
}
