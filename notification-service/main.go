package main

import (
	"log"
	"os"

	"notification-service/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Printf("[FATAL] %v", err)
		os.Exit(1)
	}
}
