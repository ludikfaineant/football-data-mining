package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"encoding/json"
	"football-data-miner/internal/db"
	"os"
)

var processedMatchesCount int

const InitialForm = 1.0

var RegularLeagues = map[int]bool{
	39:  true,
	78:  true,
	135: true,
	61:  true,
	140: true,
	144: true,
	88:  true,
	94:  true,
}

func IsRegularLeague(leagueID int) bool {
	_, exists := RegularLeagues[leagueID]
	return exists
}

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
func GetMatchStage(round string) string {
	// Классификация значений поля round
	switch {
	case strings.Contains(strings.ToLower(round), "final"):
		return "final"
	case strings.Contains(strings.ToLower(round), "semi-final") || strings.Contains(strings.ToLower(round), "semi final"):
		return "semi_final"
	case strings.Contains(strings.ToLower(round), "quarter-final") || strings.Contains(strings.ToLower(round), "quarter final"):
		return "quarter_final"
	case strings.Contains(strings.ToLower(round), "round of 16") || strings.Contains(strings.ToLower(round), "16th finals") || strings.Contains(strings.ToLower(round), "round of 8") || strings.Contains(strings.ToLower(round), "8th finals") || strings.Contains(strings.ToLower(round), "play-off"):
		return "group_stage_and_round_of_16"
	case strings.Contains(strings.ToLower(round), "group") || strings.Contains(strings.ToLower(round), "regular season") || strings.Contains(strings.ToLower(round), "league stage"):
		return "group_stage_and_round_of_16"
	default:
		return "preliminary_matches" // Значение по умолчанию
	}
}

func GetKValue(leagueID int, matchStage string) int {
	leagueIDStr := fmt.Sprintf("%d", leagueID)

	if leagueID == 2 || leagueID == 3 || leagueID == 4 || leagueID == 1 {
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

func GetPreviousForm(teamID int, leagueID int, season string, MatchDate string) float64 {
	var form float64

	err := db.DB.QueryRow(`
        SELECT COALESCE(
            (SELECT CASE 
                WHEN home_team_id = $1 THEN home_team_form 
                ELSE away_team_form 
            END AS form 
            FROM matches 
            WHERE (home_team_id = $1 OR away_team_id = $1) 
              AND league_id = $2
			  AND season = $3
              AND date < $4
            ORDER BY date DESC 
            LIMIT 1
        ),1.0)`, teamID, leagueID, season, MatchDate).Scan(&form)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Ошибка получения превформ для teamID=%d, leagueID=%d, season=%s, MatchDate=%s: %v\n",
			teamID, leagueID, season, MatchDate, err)
	}
	return form
}

func UpdateTeamForms(tx *sql.Tx, matchID int, MatchDate string, homeID, awayID int, homeScore, awayScore int, leagueID int, season string, gamma float64) error {
	var homeForm, awayForm float64
	prevHomeForm := GetPreviousForm(homeID, leagueID, season, MatchDate)
	prevAwayForm := GetPreviousForm(awayID, leagueID, season, MatchDate)

	// Обновляем форму в зависимости от результата матча
	switch {
	case homeScore > awayScore: // Победа домашней команды
		homeForm = prevHomeForm + gamma*prevAwayForm
		awayForm = prevAwayForm - gamma*prevAwayForm
	case awayScore > homeScore: // Победа гостевой команды
		homeForm = prevHomeForm - gamma*prevHomeForm
		awayForm = prevAwayForm + gamma*prevHomeForm
	default: // Ничья
		homeForm = prevHomeForm - gamma*(prevHomeForm-prevAwayForm)
		awayForm = prevAwayForm - gamma*(prevAwayForm-prevHomeForm)
	}

	// Обновляем форму в таблице matches
	_, err := tx.Exec(`
        UPDATE matches 
        SET home_team_form = $1, away_team_form = $2 
        WHERE id = $3`, homeForm, awayForm, matchID)
	if err != nil {
		return fmt.Errorf("ошибка обновления формы для матча ID=%d: %v", matchID, err)
	}

	fmt.Printf("Матч ID=%d: Форма обновлена (Home=%.2f → %.2f, Away=%.2f → %.2f)\n",
		matchID, prevHomeForm, homeForm, prevAwayForm, awayForm)
	return nil
}

func ProcessNextMatch() error {
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
	var round string
	err = db.DB.QueryRow(`
    SELECT round 
    FROM matches 
    WHERE id = $1`, matchID).Scan(&round)

	matchStage := ""
	if leagueID == 2 || leagueID == 3 || leagueID == 1 || leagueID == 4 {
		matchStage = GetMatchStage(round)
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

	if IsRegularLeague(leagueID) {
		gamma := 0.33
		if err := UpdateTeamForms(tx, matchID, matchDateStr, homeID, awayID, homeScore, awayScore, leagueID, season, gamma); err != nil {
			return fmt.Errorf("ошибка обновления формы для матча ID=%d: %v", matchID, err)
		}
	} else {
		fmt.Printf("Матч ID=%d: Не регулярный чемпионат. Форма не обновляется.\n", matchID)
	}

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
			if err == sql.ErrNoRows {
				fmt.Println("Программа завершена: все матчи обработаны.")
				break
			}

			log.Fatalf("Ошибка: %v", err)
		}
	}
}
