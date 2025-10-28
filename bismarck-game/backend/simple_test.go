package main

import (
	"encoding/json"
	"fmt"
	"log"

	"bismarck-game/backend/internal/config"
	"bismarck-game/backend/pkg/database"
)

func main() {
	fmt.Println("🚢 Simple Task Force Test")
	fmt.Println("=========================")

	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 1. Проверим структуру таблицы task_forces
	fmt.Println("\n📋 1. Проверка структуры таблицы task_forces...")

	rows, err := db.Query("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'task_forces' ORDER BY ordinal_position")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Columns:")
	for rows.Next() {
		var col, typ string
		rows.Scan(&col, &typ)
		fmt.Printf("  ✓ %s: %s\n", col, typ)
	}

	// 2. Проверим что поле task_force_id добавлено в naval_units
	fmt.Println("\n🔗 2. Проверка связи naval_units -> task_forces...")

	var hasColumn bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'naval_units' AND column_name = 'task_force_id')").Scan(&hasColumn)
	if err != nil {
		log.Fatal(err)
	}

	if hasColumn {
		fmt.Println("  ✅ Поле task_force_id найдено в naval_units")
	} else {
		fmt.Println("  ❌ Поле task_force_id НЕ найдено в naval_units")
	}

	// 3. Создадим тестовый Task Force напрямую в БД
	fmt.Println("\n🎯 3. Создание тестового Task Force...")

	taskForceID := "tf-test-123"
	gameID := "game-test-456"

	unitsJSON, _ := json.Marshal([]string{"unit1", "unit2"})

	_, err = db.Exec(`
		INSERT INTO task_forces (id, game_id, name, owner, nationality, position, speed, units, is_visible, detection_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name,
			updated_at = CURRENT_TIMESTAMP
	`, taskForceID, gameID, "TF-1", "player1", "allied", "A1", 3, unitsJSON, true, "none")

	if err != nil {
		log.Printf("❌ Ошибка создания Task Force: %v", err)
	} else {
		fmt.Println("  ✅ Task Force создан успешно")
	}

	// 4. Проверим автоматическое именование (constraint)
	fmt.Println("\n📛 4. Проверка constraint на именование...")

	_, err = db.Exec(`
		INSERT INTO task_forces (game_id, name, owner, nationality, position, speed, units)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, gameID, "INVALID-NAME", "player1", "german", "B2", 2, unitsJSON)

	if err != nil {
		fmt.Println("  ✅ Constraint работает - неправильное имя отклонено:")
		fmt.Printf("     %v\n", err)
	} else {
		fmt.Println("  ❌ Constraint НЕ работает - неправильное имя принято")
	}

	// 5. Создадим Task Force с правильным именем
	fmt.Println("\n✅ 5. Создание Task Force с правильным именем...")

	_, err = db.Exec(`
		INSERT INTO task_forces (game_id, name, owner, nationality, position, speed, units)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, gameID, "KG-1", "player2", "german", "C3", 4, unitsJSON)

	if err != nil {
		log.Printf("❌ Ошибка: %v", err)
	} else {
		fmt.Println("  ✅ Task Force KG-1 создан успешно")
	}

	// 6. Получим все Task Forces
	fmt.Println("\n📊 6. Получение всех Task Forces...")

	rows, err = db.Query("SELECT name, nationality, position, speed FROM task_forces WHERE game_id = $1", gameID)
	if err != nil {
		log.Printf("❌ Ошибка получения: %v", err)
	} else {
		defer rows.Close()
		fmt.Println("Найденные Task Forces:")
		for rows.Next() {
			var name, nationality, position string
			var speed int
			rows.Scan(&name, &nationality, &position, &speed)
			fmt.Printf("  🚢 %s (%s) at %s, speed %d\n", name, nationality, position, speed)
		}
	}

	// Очистим тестовые данные
	fmt.Println("\n🧹 Очистка тестовых данных...")
	_, err = db.Exec("DELETE FROM task_forces WHERE game_id = $1", gameID)
	if err != nil {
		log.Printf("Warning: failed to cleanup: %v", err)
	} else {
		fmt.Println("  ✅ Тестовые данные очищены")
	}

	fmt.Println("\n🎉 Тест завершен успешно!")
}
