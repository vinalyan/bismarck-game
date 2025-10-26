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

	rows, err := db.Query("SELECT username, email FROM users ORDER BY created_at DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Пользователи в БД:")
	for rows.Next() {
		var username, email string
		rows.Scan(&username, &email)
		fmt.Printf("- %s (%s)\n", username, email)
	}
}
