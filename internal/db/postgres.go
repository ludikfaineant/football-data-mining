package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"football-data-miner/internal/models"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"))
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", err)
	}

	log.Println("Успешное подключение к БД")
}
func CloseDB() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("Ошибка при закрытии подключения к БД: %v", err)
		} else {
			log.Println("Подключение к БД успешно закрыто")
		}
	}
}
func GetNextUnprocessedSeason() (int, string) {
	var leagueID int
	var season string
	err := DB.QueryRowContext(context.Background(), `
        SELECT league_id, season
        FROM league_seasons
        WHERE is_processed = FALSE
        ORDER BY season ASC
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

func MarkSeasonAsProcessed(leagueID int, season string) error {
	query := `
        UPDATE league_seasons
        SET is_processed = TRUE
        WHERE league_id = $1 AND season = $2
    `
	_, err := DB.Exec(query, leagueID, season)
	return err
}

func GetProcessedSeasons() ([]models.Season, error) {
	query := `
        SELECT league_id, season
        FROM league_seasons
        WHERE is_processed = TRUE
    `
	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса обработанных сезонов: %v", err)
	}
	defer rows.Close()

	var seasons []models.Season
	for rows.Next() {
		var season models.Season
		err := rows.Scan(&season.LeagueID, &season.Season)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования сезона: %v", err)
		}
		seasons = append(seasons, season)
	}

	return seasons, nil
}
func IsMatchExists(matchID int) (bool, error) {
	query := `
        SELECT EXISTS (
            SELECT 1
            FROM matches
            WHERE id = $1
        )
    `
	var exists bool
	err := DB.QueryRow(query, matchID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки существования матча ID=%d: %v", matchID, err)
	}
	return exists, nil
}

func GetSeasonMatches(leagueID int, seasonDate string) ([]models.Match, error) {
	query := `
        SELECT id, date, home_team_id, away_team_id, home_score, away_score 
        FROM matches 
        WHERE league_id = $1 AND date <= $2 
        ORDER BY date ASC
    `
	rows, err := DB.Query(query, leagueID, seasonDate)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения матчей сезона: %v", err)
	}
	defer rows.Close()

	var matches []models.Match
	for rows.Next() {
		var match models.Match
		err := rows.Scan(&match.ID, &match.Date, &match.HomeTeamID, &match.AwayTeamID, &match.HomeScore, &match.AwayScore)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования матча: %v", err)
		}
		matches = append(matches, match)
	}

	return matches, nil
}
func GetMissingMatchesFromDB(db *sql.DB) ([]models.Match, error) {
	query := `
        SELECT id, date, league_id, season, home_team_id, away_team_id, home_score, away_score
        FROM matches
        WHERE season >= '2018'
          AND league_id IN (39, 78, 135, 140, 61)
          AND id NOT IN (SELECT match_id FROM match_statistics)
    `
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer rows.Close()

	var matches []models.Match
	for rows.Next() {
		var match models.Match
		var leagueID int
		var season string
		err := rows.Scan(
			&match.ID,
			&match.Date,
			&leagueID,
			&season,
			&match.HomeTeamID,
			&match.AwayTeamID,
			&match.HomeScore,
			&match.AwayScore,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования матча: %v", err)
		}
		matches = append(matches, match)
	}

	return matches, nil
}

func GetLeagueAndSeasonForMatch(matchID int) (int, string, error) {
	query := `
        SELECT league_id, season
        FROM matches
        WHERE id = $1
    `
	var leagueID int
	var season string
	err := DB.QueryRow(query, matchID).Scan(&leagueID, &season)
	if err != nil {
		return 0, "", fmt.Errorf("ошибка получения лиги и сезона для матча ID=%d: %v", matchID, err)
	}
	return leagueID, season, nil
}

func GetProcessedMatches(leagueID int, season string) ([]int, error) {
	query := `
        SELECT id 
        FROM matches
        WHERE league_id=$1 and season=$2
    `
	rows, err := DB.Query(query, leagueID, season)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer rows.Close()

	var matchIDs []int
	for rows.Next() {
		var matchID int
		if err := rows.Scan(&matchID); err != nil {
			return nil, fmt.Errorf("ошибка сканирования ID матча: %v", err)
		}
		matchIDs = append(matchIDs, matchID)
	}
	return matchIDs, nil
}
