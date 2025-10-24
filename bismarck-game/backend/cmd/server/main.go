// @title Bismarck Game API
// @version 1.0
// @description API для игры Bismarck Chase - пошаговой стратегической игры о морских сражениях
// @contact.name Bismarck Game Team
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description JWT токен в формате: Bearer {token}
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
