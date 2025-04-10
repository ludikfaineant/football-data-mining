package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Импортируем драйвер PostgreSQL
)

var dbConn *sql.DB

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения системы")
	}
}

func InitDB() {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"))
	var err error
	dbConn, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	err = dbConn.Ping()
	if err != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", err)
	}

	log.Println("Успешное подключение к БД")
}

func GetNextUnprocessedSeason() (int, string) {
	var leagueID int
	var season string
	err := dbConn.QueryRowContext(context.Background(), `
        SELECT league_id, season
        FROM league_seasons
        WHERE is_processed = FALSE
        ORDER BY season DESC
        LIMIT 1
    `).Scan(&leagueID, &season)

	if err == sql.ErrNoRows {
		return 0, ""
	}
	if err != nil {
		panic(err)
	}

	return leagueID, season
}
