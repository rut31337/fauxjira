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
	RegisterIssueRoutes(mux, db)
	RegisterSearchRoutes(mux, db)
	RegisterUserRoutes(mux, db)
	RegisterAdminRoutes(mux, db, cfg)

	addr := ":" + cfg.Port
	fmt.Printf("fauxjira listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
