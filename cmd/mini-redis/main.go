package main

import (
	"fmt"
	"mini-redis/internal"
)

func main() {
	fmt.Println("🚀 Starting MiniRedis with AOF...")

	aof, err := internal.NewAOF("appendonly.aof")
	if err != nil {
		fmt.Println("❌ Error initializing AOF:", err)
		return
	}
	defer aof.Close()

	store := internal.NewStore(aof)
	server := internal.NewServer(":6380", store)

	if err := server.Start(); err != nil {
		fmt.Println("❌ Server error:", err)
	}
}
