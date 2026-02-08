package main

import (
	"fmt"
	"os"

	"warreth.dev/immich-sync/pkg/app"
	"warreth.dev/immich-sync/pkg/config"
)

func main() {
	fmt.Println(">> Immich Sync Tool <<")

	cfg, err := config.ReadConfig("config.json")
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		if os.Getenv("IMMICH_API_KEY") == "" {
			fmt.Println("Please provide config.json or environment variables.")
			os.Exit(1)
		}
	}

	application, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing app: %v\n", err)
		os.Exit(1)
	}

	application.Run()
}
