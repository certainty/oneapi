package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/certainty/oneapi/internal/server"
	"github.com/certainty/oneapi/internal/spec"
	"github.com/certainty/oneapi/internal/storage"
)

var (
	flagDebug bool
)

func main() {
	flag.BoolVar(&flagDebug, "debug", false, "enable debug mode")

	flag.Usage = func() {
		fmt.Printf("Usage: %s [flags] manifest-path\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(os.Args) != 2 {
		flag.Usage()
	}
	manifest, err := spec.LoadManifest(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to load manifest: %v", err)
	}

	db, err := storage.NewSQLiteDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	repositories := make(map[string]storage.Repository)
	for entityName, entityDef := range manifest.Entities {
		entityRegistry := storage.NewEntityRegistry()
		entityObj := entityRegistry.RegisterEntity(entityName, entityDef)

		// Create repository for entity
		repo := storage.NewSQLiteRepository(db, entityObj)
		repositories[entityName] = repo

		// Create database schema for entity
		if err := repo.CreateSchema(); err != nil {
			log.Fatalf("Failed to create schema for entity %s: %v", entityName, err)
		}
	}

	serverOpts, err := server.OptionsFromManifest(*manifest)
	if err != nil {
		log.Fatalf("Failed to get server options: %v", err)
	}

	srv := server.NewServer(*serverOpts, repositories)
	go srv.Start()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down server...")
}
