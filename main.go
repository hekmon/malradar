package main

import (
	"context"
	"os"

	"github.com/hekmon/hllogger"
	"github.com/hekmon/malwatcher/mal"
)

var (
	logger        *hllogger.HlLogger
	mainLock      chan struct{}
	mainCtx       context.Context
	mainCtxCancel func()
	watcher       *mal.Controller
)

func main() {
	logger = hllogger.New(os.Stdout, &hllogger.Config{
		LogLevel: hllogger.Debug,
	})
	mainCtx, mainCtxCancel = context.WithCancel(context.Background())
	defer mainCtxCancel()
	watcher = mal.New(mainCtx, mal.Config{
		NbSeasons: 8,
		Logger:    logger,
	})
	if watcher == nil {
		logger.Fatal(1, "[Main] Failted to instanciate the watcher")
	}
	mainLock = make(chan struct{})
	go handleSignals()
	<-mainLock
}
