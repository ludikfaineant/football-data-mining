package api

import (
	"encoding/json"
	"fmt"
	"football-data-miner/internal/models"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var lastRequestTime time.Time = time.Now()

func FetchStatistics(fixtureID int) ([]TeamStatistics, bool, error) {
	endpoint := fmt.Sprintf("%s/fixtures/statistics?fixture=%d", os.Getenv("API_BASE_URL"), fixtureID)
	resp, err := makeRequest(endpoint)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	var statsResponse struct {
		Response []TeamStatistics `json:"response"`
	}
	remainingStr := resp.Header.Get("x-ratelimit-requests-remaining")
	remaining, err := strconv.Atoi(remainingStr)
	if err != nil {
		return []TeamStatistics{}, false, fmt.Errorf("Ошибка парсинга x-ratelimit-requests-remaining: %v", err)
	}
	if remaining < 4 {
		fmt.Println("Меньше 4 запросов осталось. Завершаем обработку")
		return statsResponse.Response, false, nil
	}

	err = json.NewDecoder(resp.Body).Decode(&statsResponse)
	return statsResponse.Response, true, err
}

func FetchLineups(fixtureID int) (LineupResponse, error) {
	endpoint := fmt.Sprintf("%s/fixtures/lineups?fixture=%d", os.Getenv("API_BASE_URL"), fixtureID)
	resp, err := makeRequest(endpoint)
	if err != nil {
		return LineupResponse{}, err
	}
	defer resp.Body.Close()

	var lineups LineupResponse
	err = json.NewDecoder(resp.Body).Decode(&lineups)
	return lineups, err
}

func FetchPlayers(fixtureID int) (PlayersResponse, error) {
	var players PlayersResponse
	endpoint := fmt.Sprintf("%s/fixtures/players?fixture=%d", os.Getenv("API_BASE_URL"), fixtureID)
	resp, err := makeRequest(endpoint)
	if err != nil {
		return PlayersResponse{}, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&players)
	if err != nil {
		return PlayersResponse{}, fmt.Errorf("Ошибка декодирования данных: %v", err)
	}

	return players, nil
}

func FetchSeasonMatches(leagueID int, season string) ([]models.Match, error) {
	endpoint := fmt.Sprintf("%s/fixtures?league=%d&season=%s", os.Getenv("API_BASE_URL"), leagueID, season)
	resp, err := makeRequest(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var matchesResponse models.MatchesOfSeason
	err = json.NewDecoder(resp.Body).Decode(&matchesResponse)
	if err != nil {
		return nil, fmt.Errorf("ошибка декодирования JSON: %v", err)
	}

	var matches []models.Match
	for _, m := range matchesResponse.Response {
		matches = append(matches, models.Match{
			ID:            m.Fixture.ID,
			Date:          m.Fixture.Date,
			HomeTeamID:    m.Teams.Home.ID,
			AwayTeamID:    m.Teams.Away.ID,
			HomeTeamName:  m.Teams.Home.Name,
			AwayTeamName:  m.Teams.Away.Name,
			HomeScore:     safeInt(m.Goals.Home),
			AwayScore:     safeInt(m.Goals.Away),
			HomeCoachID:   0,
			AwayCoachID:   0,
			HomeFormation: "",
			AwayFormation: "",
		})
	}

	return matches, nil
}
func makeRequest(url string) (*http.Response, error) {
	time.Sleep(300 * time.Millisecond)
	//time.Sleep(3 * time.Second)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании запроса: %v", err)
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY не установлен")
	}

	req.Header.Set("X-RapidAPI-Key", apiKey)
	req.Header.Set("X-RapidAPI-Host", "v3.football.api-sports.io")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("API вернул статус %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func safeInt(value interface{}) int {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		numStr := strings.TrimSuffix(v, "%")
		num, _ := strconv.Atoi(numStr)
		return num
	default:
		return 0
	}
}
