package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	dsn := "host=localhost port=5432 user=bismarck_user password=bismarck_pass dbname=bismarck_game_dev sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT username, email, id FROM users ORDER BY created_at DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Пользователи в БД:")
	fmt.Println("====================================")
	for rows.Next() {
		var username, email, id string
		rows.Scan(&username, &email, &id)
		fmt.Printf("✅ %s (%s) - ID: %s\n", username, email, id)
	}
	fmt.Println("====================================")
}
