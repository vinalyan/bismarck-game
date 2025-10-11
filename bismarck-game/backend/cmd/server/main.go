package main

import (
	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/internal/server"
	"log"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// Создание и запуск сервера
	srv := server.New(cfg)

	log.Printf("Starting Bismarck Game Server on %s", cfg.Server.Address)
	log.Printf("Game settings: %d players, %v turn duration",
		cfg.Game.MaxPlayers, cfg.Game.TurnDuration.ToDuration())

	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
