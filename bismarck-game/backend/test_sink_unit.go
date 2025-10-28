package main

import (
	"fmt"
	"log"

	"bismarck-game/backend/internal/game/services"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/testutil"
)

func main() {
	// Подключаемся к БД
	db, err := testutil.SetupTestDatabase()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Создаем сервисы
	unitLogger, _ := logger.New(logger.INFO, "unit-service", "stdout")
	unitService := services.NewUnitService(db, unitLogger)

	taskForceLogger, _ := logger.New(logger.INFO, "taskforce-service", "stdout")
	movementService := services.NewMovementService(db, unitLogger, nil, nil, unitService, nil, nil)
	taskForceService := services.NewTaskForceService(db, taskForceLogger, unitService, movementService)

	// Настраиваем callback
	unitService.SetUnitSunkHandler(taskForceService.HandleUnitSunk)

	// Потопим корабль через сервис
	unitID := "78320ee4-154e-4408-b30d-ed3679e1e109" // PRINZ EUGEN
	fmt.Printf("Потопление корабля %s...\n", unitID)

	err = unitService.DeleteNavalUnit(unitID)
	if err != nil {
		log.Fatal("Failed to sink unit:", err)
	}

	fmt.Println("Корабль потоплен! Проверьте Task Force.")
}
