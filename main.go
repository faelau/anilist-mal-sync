package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
)

const debug = false

var (
	configFile = flag.String("c", "config.yaml", "path to config file")
	forceSync  = flag.Bool("f", false, "force sync all animes")
	dryRun     = flag.Bool("d", false, "dry run without updating MyAnimeList")
	mangaSync  = flag.Bool("manga", false, "sync manga instead of anime")
)

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	config, err := loadConfigFromFile(*configFile)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	app, err := NewApp(ctx, config, *forceSync, *dryRun, *mangaSync)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
