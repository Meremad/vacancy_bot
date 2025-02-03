package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitializeDB инициализирует базу данных
func InitializeDB() {
	var err error
	DB, err = sql.Open("sqlite3", "bot_data.db")
	if err != nil {
		log.Fatalf("Ошибка при открытии базы данных: %v", err)
	}

	// Создание таблицы пользователей
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		telegram_id INTEGER NOT NULL UNIQUE,
		language TEXT NOT NULL
	);
	`
	_, err = DB.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Ошибка при создании таблицы: %v", err)
	}
}

// SaveUserPreferences сохраняет предпочтения пользователя
func SaveUserPreferences(telegramID int64, language string) {
	query := `
	INSERT INTO users (telegram_id, language)
	VALUES (?, ?)
	ON CONFLICT(telegram_id) DO UPDATE SET language = excluded.language;
	`
	_, err := DB.Exec(query, telegramID, language)
	if err != nil {
		log.Printf("Ошибка сохранения предпочтений: %v", err)
	}
}

// GetUserPreferences получает предпочтения пользователя
func GetUserPreferences(telegramID int64) string {
	query := `SELECT language FROM users WHERE telegram_id = ?`
	row := DB.QueryRow(query, telegramID)

	var language string
	err := row.Scan(&language)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Golang" // Язык по умолчанию
		}
		log.Printf("Ошибка получения предпочтений: %v", err)
		return ""
	}

	return language
}
