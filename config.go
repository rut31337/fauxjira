package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

type Config struct {
	Port          string
	DBPath        string
	AdminPassword string
	UserPassword  string
}

func LoadConfig() Config {
	c := Config{
		Port:          envOrDefault("FAUXJIRA_PORT", "6778"),
		DBPath:        envOrDefault("FAUXJIRA_DB_PATH", "fauxjira.db"),
		AdminPassword: os.Getenv("FAUXJIRA_ADMIN_PASSWORD"),
		UserPassword:  os.Getenv("FAUXJIRA_USER_PASSWORD"),
	}
	if c.AdminPassword == "" {
		c.AdminPassword = randomPassword()
		fmt.Printf("Generated admin password: %s\n", c.AdminPassword)
	}
	if c.UserPassword == "" {
		c.UserPassword = randomPassword()
		fmt.Printf("Generated user password: %s\n", c.UserPassword)
	}
	return c
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func randomPassword() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}
