package main

import (
	"os"
	"os/signal"
	"syscall"
)

func handleSignals() {
	// If we exit, allow main goroutine to do so
	defer close(mainLock)
	// Register signals
	var sig os.Signal
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	// Waiting for signals to catch
	for {
		sig = <-signalChannel
		switch sig {
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGINT:
			logger.Infof("[Main] Signal '%v' caught: cleaning up before exiting", sig)
			// if err := systemd.NotifyStopping(); err != nil {
			// 	logger.Errorf("[Main] can't send systemd stopping notification: %v", err)
			// } else {
			// 	logger.Debug("[Main] systemd stopping notification sent")
			// }
			// pushoverClient.SendHighPriorityMsg("Application is stopping...", "", "signals: stopping")
			// Cancel main ctx & wait for watcher
			mainCtxCancel()
			watcher.WaitStopped()
			logger.Debugf("[Main] Signal '%v' caught: watcher stopped: unlocking main goroutine to exit", sig)
			return
		default:
			logger.Warningf("[Main] Signal '%v' caught but no process set to handle it: skipping", sig)
		}
	}
}
