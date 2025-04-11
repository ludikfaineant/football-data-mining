package main

import (
	"fmt"
	"football-data-miner/internal/api"
	"football-data-miner/internal/cache"
	"football-data-miner/internal/db"
	"football-data-miner/internal/models"
	"strconv"
	"strings"
)

func main() {
	db.InitDB()
	cache.InitRedis()

	if cache.IsCacheEmpty() {
		fmt.Println("Кэш пустой. Берем следующий сезон из БД...")
		leagueID, season := db.GetNextUnprocessedSeason()
		if leagueID == 0 {
			fmt.Println("Все сезоны обработаны!")
			return
		}

		fmt.Printf("Обрабатываем сезон: лига %d, сезон %s\n", leagueID, season)
		matches, err := api.FetchSeasonMatches(leagueID, season)
		if err != nil {
			fmt.Printf("Ошибка при получении матчей: %v\n", err)
			return
		}

		err = cache.CacheSeasonMatches(leagueID, season, matches)
		if err != nil {
			fmt.Printf("Ошибка при сохранении матчей в Redis: %v\n", err)
			return
		}

		fmt.Println("Матчи успешно сохранены в Redis. Начинаем обработку...")
		processMatches(leagueID, season, matches)
	} else {
		fmt.Println("Кэш содержит матчи. Продолжаем обработку...")
		processCachedMatches()
	}
}

func processMatches(leagueID int, season string, matches []models.Match) {
	for _, match := range matches {

		processed, err := cache.IsMatchProcessed(leagueID, season, match.ID)
		if err != nil || processed {
			continue
		}

		err = db.SaveTeamIfNotExists(match.HomeTeamID, match.HomeTeamName)
		if err != nil {
			fmt.Printf("Ошибка сохранения команды %d: %v\n", match.HomeTeamID, err)
			continue
		}
		err = db.SaveTeamIfNotExists(match.AwayTeamID, match.AwayTeamName)
		if err != nil {
			fmt.Printf("Ошибка сохранения команды %d: %v\n", match.AwayTeamID, err)
			continue
		}

		stats, err := api.FetchStatistics(match.ID)
		if err != nil {
			fmt.Printf("Ошибка статистики: %v\n", err)
			continue
		}

		lineups, err := api.FetchLineups(match.ID)
		if err != nil {
			fmt.Printf("Ошибка составов: %v\n", err)
			continue
		}

		players, canContinue, err := api.FetchPlayers(match.ID)
		if err != nil {
			fmt.Printf("Ошибка событий: %v\n", err)
			continue
		}

		parsedStats, _ := api.ParseStatistics(match.ID, stats)
		parsedLineups := api.MergeLineupAndPlayers(lineups, players, &match)

		db.SaveMatchDetails(match, leagueID, season, parsedStats, parsedLineups)
		cache.MarkMatchAsProcessed(leagueID, season, match.ID)
		if !canContinue {
			break
		}
	}
}
func processCachedMatches() {
	keys, _ := cache.GetAllSeasonKeys()
	for _, key := range keys {
		leagueID, season := parseLeagueAndSeasonFromKey(key)
		matches, err := cache.GetSeasonMatches(leagueID, season)
		if err != nil {
			fmt.Printf("Ошибка при получении матчей: %v\n", err)
			continue
		}
		processMatches(leagueID, season, matches)
	}
}

func parseLeagueAndSeasonFromKey(key string) (int, string) {
	parts := strings.Split(key, ":")
	leagueID, _ := strconv.Atoi(parts[2])
	return leagueID, parts[3]
}
