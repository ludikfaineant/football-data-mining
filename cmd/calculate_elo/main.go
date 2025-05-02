package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	"encoding/json"
	"football-data-miner/internal/db"
	"os"
)

var processedMatchesCount int

type EloConfig struct {
	TournamentWeights     map[string]map[string]int `json:"tournament_weights"`
	NationalLeagueWeights map[string][]string       `json:"national_league_weights"`
	KValues               map[string]int            `json:"k_values"`
	InitialRatings        map[string]int            `json:"initial_ratings"`
}

var eloConfig EloConfig

func LoadEloConfig(filePath string) error {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	err = json.Unmarshal(file, &eloConfig)
	if err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	return nil
}

func ParseDate(dateStr string) time.Time {
	layout := "2006-01-02T15:04:05Z"
	t, err := time.Parse(layout, dateStr)
	if err != nil {
		log.Fatalf("Ошибка парсинга даты: %v", err)
	}
	return t
}

func CalculateGoalFactor(goalDifference int) float64 {
	if goalDifference < 0 {
		goalDifference = -goalDifference
	}
	switch {
	case goalDifference == 0 || goalDifference == 1:
		return 1
	case goalDifference == 2:
		return 1.5
	default:
		return float64(11+goalDifference) / 8
	}
}

func GetLeagueType(leagueID int) string {
	leagueIDStr := fmt.Sprintf("%d", leagueID)

	for category, leagues := range eloConfig.NationalLeagueWeights {
		for _, id := range leagues {
			if id == leagueIDStr {
				return category
			}
		}
	}

	return "other"
}

func GetMatchStage(matchDate string, leagueID int, season string) string {
	parsedDate, _ := time.Parse("2006-01-02", matchDate[:10])

	var positionFromEnd int
	err := db.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM matches 
        WHERE league_id = $1 AND season = $2`,
		leagueID, season).Scan(&positionFromEnd)
	if err != nil {
		log.Fatalf("Ошибка подсчета матчей: %v", err)
	}

	switch {
	case positionFromEnd == 1:
		return "final"
	case positionFromEnd <= 5:
		return "semi_final"
	case positionFromEnd <= 13:
		return "quarter_final"
	case parsedDate.Month() >= time.September:
		return "group_stage_and_round_of_16"
	default:
		return "preliminary_matches"
	}
}
func GetKValue(leagueID int, matchStage string) int {
	leagueIDStr := fmt.Sprintf("%d", leagueID)

	if leagueID == 2 || leagueID == 3 {
		if tournamentWeights, exists := eloConfig.TournamentWeights[leagueIDStr]; exists {
			if k, ok := tournamentWeights[matchStage]; ok {
				return k
			}
		}
		return 5
	}

	category := GetLeagueType(leagueID)
	return eloConfig.KValues[category]
}
func GetInitialRating(leagueID int) int {
	leagueIDStr := fmt.Sprintf("%d", leagueID)

	if rating, exists := eloConfig.InitialRatings[leagueIDStr]; exists {
		return rating
	}

	return eloConfig.InitialRatings["other"]
}

func GetPreviousElo(teamID int, currentMatchDate string, leagueID int) int {
	var elo int

	err := db.DB.QueryRow(`
        SELECT COALESCE(
            (SELECT CASE 
                WHEN home_team_id = $1 THEN home_team_elo 
                ELSE away_team_elo 
            END AS elo 
            FROM matches 
            WHERE (home_team_id = $1 OR away_team_id = $1) 
              AND date < $2 
            ORDER BY date DESC 
            LIMIT 1
        ), $3)`, teamID, currentMatchDate, GetInitialRating(leagueID)).Scan(&elo)
	if err != nil {
		log.Fatalf("Ошибка получения Elo для команды ID=%d: %v", teamID, err)
	}

	return elo
}
func CalculateElo(homeElo, awayElo, homeScore, awayScore int, kFactor float64) (int, int) {
	dr := float64(homeElo-awayElo) + 100

	expectedHome := 1 / (1 + math.Pow(10, -dr/400))
	expectedAway := 1 - expectedHome

	resultHome, resultAway := 0.5, 0.5
	switch {
	case homeScore > awayScore:
		resultHome, resultAway = 1, 0
	case homeScore < awayScore:
		resultHome, resultAway = 0, 1
	}

	goalDifference := homeScore - awayScore
	goalFactor := CalculateGoalFactor(goalDifference)

	newHome := homeElo + int(kFactor*goalFactor*(resultHome-expectedHome))
	newAway := awayElo + int(kFactor*goalFactor*(resultAway-expectedAway))
	return newHome, newAway
}

func ProcessNextMatch() error {
	/*if processedMatchesCount >= 1000 {
		fmt.Println("Тест завершен: обработано 1000 матчей.")
		time.Sleep(time.Second)
		return nil
	}*/
	var matchID, homeID, awayID, homeScore, awayScore int
	var matchDateStr, season string
	var leagueID int

	err := db.DB.QueryRow(`
        SELECT id, date, home_team_id, away_team_id, home_score, away_score, league_id, season
        FROM matches 
        WHERE home_team_elo IS NULL OR away_team_elo IS NULL 
        ORDER BY date ASC 
        LIMIT 1`).Scan(&matchID, &matchDateStr, &homeID, &awayID, &homeScore, &awayScore, &leagueID, &season)
	if err == sql.ErrNoRows {
		fmt.Println("Все матчи обработаны!")
		return nil
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("ошибка транзакции: %v", err)
	}
	defer tx.Rollback()

	matchStage := ""
	if leagueID == 2 || leagueID == 3 {
		matchStage = GetMatchStage(matchDateStr, leagueID, season)
	}

	homeElo := GetPreviousElo(homeID, matchDateStr, leagueID)
	awayElo := GetPreviousElo(awayID, matchDateStr, leagueID)

	kFactor := GetKValue(leagueID, matchStage)
	newHomeElo, newAwayElo := CalculateElo(homeElo, awayElo, homeScore, awayScore, float64(kFactor))

	_, err = tx.Exec(`
        UPDATE matches 
        SET home_team_elo = $1, away_team_elo = $2 
        WHERE id = $3`, newHomeElo, newAwayElo, matchID)
	if err != nil {
		return fmt.Errorf("ошибка обновления Elo для матча ID=%d: %v", matchID, err)
	}

	fmt.Printf("Матч ID=%d: Elo обновлены (Home=%d → %d, Away=%d → %d)\n",
		matchID, homeElo, newHomeElo, awayElo, newAwayElo)
	processedMatchesCount++
	return tx.Commit()
}

func main() {
	fmt.Print(os.Getwd())
	if err := LoadEloConfig("./cmd/calculate_elo/elo_config.json"); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	db.InitDB()
	defer db.CloseDB()

	for {
		err := ProcessNextMatch()
		if err != nil {
			log.Fatalf("Ошибка: %v", err)
		}
	}
}
