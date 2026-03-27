package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg := LoadConfig()

	db, err := InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if err := SeedUsers(db, cfg); err != nil {
		log.Fatalf("Failed to seed users: %v", err)
	}

	mux := http.NewServeMux()
	// Routes will be registered in subsequent tasks

	addr := ":" + cfg.Port
	fmt.Printf("fauxjira listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
