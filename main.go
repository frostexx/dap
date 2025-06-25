package main

import (
	"fmt"
	"log"
	"os"
	"pi/server"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

func loadConfig() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("error loading env config: %v", err)
	}
	color.Green("loaded base env config")

	network := os.Getenv("NETWORK")
	if network != "mainnet" && network != "testnet" {
		return fmt.Errorf("invalid network")
	}
	fmt.Println(network)

	rConfig := ".env." + network
	err = godotenv.Overload(rConfig)
	if err != nil {
		return fmt.Errorf("error loading network specific config")
	}
	color.GreenString("loaded %s config", network)

	return nil
}

func main() {
	err := loadConfig()
	if err != nil {
		log.Fatalf("%s: %v", "config", err)
	}

	srv := server.New()
	err = srv.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}
