package main

import (
	"context"
	"fmt"
	"football-data-miner/internal/api"
	"football-data-miner/internal/cache"
	"football-data-miner/internal/db"
	"football-data-miner/internal/models"
	"strconv"
	"strings"
)

func main() {
	var shouldExit bool
	db.InitDB()
	defer db.CloseDB()

	cache.InitRedis()
	//fmt.Println("Проверка недостающих матчей...")
	//RecheckAndCacheMissingMatches()

	for {
		if cache.IsCacheEmpty() {
			fmt.Println("Кэш пустой. Берем следующий сезон из БД...")
			leagueID, season := db.GetNextUnprocessedSeason()
			if leagueID == 0 {
				fmt.Println("Все сезоны обработаны!")
				break
			}

			fmt.Printf("Обрабатываем сезон: лига %d, сезон %s\n", leagueID, season)
			matches, err := api.FetchSeasonMatches(leagueID, season)
			if err != nil {
				fmt.Printf("Ошибка при получении матчей: %v\n", err)
				continue
			}

			err = cache.CacheSeasonMatches(leagueID, season, matches)
			if err != nil {
				fmt.Printf("Ошибка при сохранении матчей в Redis: %v\n", err)
				continue
			}

			fmt.Println("Матчи успешно сохранены в Redis. Начинаем обработку...")
			shouldExit = processMatches(leagueID, season, matches)
		} else {
			fmt.Println("Кэш содержит матчи. Продолжаем обработку...")
			shouldExit = processCachedMatches()
		}
		if shouldExit {
			break
		}
	}
}

func processMatches(leagueID int, season string, matches []models.Match) bool {
	totalMatches := len(matches)

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

		stats, canContinue, err := api.FetchStatistics(match.ID)
		if err != nil {
			fmt.Printf("Ошибка статистики: %v\n", err)
			continue
		}
		/*
			lineups, err := api.FetchLineups(match.ID)
			if err != nil {
				fmt.Printf("Ошибка составов: %v\n", err)
				continue
			}

			players, err := api.FetchPlayers(match.ID)
			if err != nil {
				fmt.Printf("Ошибка событий: %v\n", err)
				continue
			}
		*/
		parsedStats, _ := api.ParseStatistics(match.ID, stats)
		//		parsedLineups := api.MergeLineupAndPlayers(lineups, players, &match)
		db.SaveMatchDetails(match, leagueID, season, parsedStats, nil)
		//db.SaveMatchDetails(match, leagueID, season, parsedStats, parsedLineups)
		cache.MarkMatchAsProcessed(leagueID, season, match.ID)
		if !canContinue {
			return true
		}
		isCompleted, err := cache.IsSeasonCompleted(leagueID, season, totalMatches)
		if isCompleted {
			fmt.Printf("Сезон лиги %d, сезон %s завершен!\n", leagueID, season)
			cleanupSeason(leagueID, season)
			return false
		}
	}
	return false
}
func processCachedMatches() bool {
	keys, _ := cache.GetAllSeasonKeys()
	for _, key := range keys {
		leagueID, season := parseLeagueAndSeasonFromKey(key)
		matches, err := cache.GetSeasonMatches(leagueID, season)
		if err != nil {
			fmt.Printf("Ошибка при получении матчей: %v\n", err)
			continue
		}
		return processMatches(leagueID, season, matches)
	}
	return false
}

func parseLeagueAndSeasonFromKey(key string) (int, string) {
	parts := strings.Split(key, ":")
	leagueID, _ := strconv.Atoi(parts[2])
	return leagueID, parts[3]
}

func cleanupSeason(leagueID int, season string) {
	cacheKey := cache.GetSeasonKey(leagueID, season)
	processedKey := cache.GetProcessedKey(leagueID, season)

	err := cache.Rdb.Del(context.Background(), cacheKey, processedKey).Err()
	if err != nil {
		fmt.Printf("Ошибка очистки кэша: %v\n", err)
	} else {
		fmt.Println("Кэш очищен.")
	}
	err = db.MarkSeasonAsProcessed(leagueID, season)
}
func RecheckAndCacheMissingMatches() {
	processedSeasons, err := db.GetProcessedSeasons()
	if err != nil {
		fmt.Printf("Ошибка получения обработанных сезонов: %v\n", err)
		return
	}

	for _, season := range processedSeasons {
		leagueID := season.LeagueID
		seasonName := season.Season

		fmt.Printf("Проверяем сезон: лига %d, сезон %s\n", leagueID, seasonName)

		matches, err := api.FetchSeasonMatches(leagueID, seasonName)
		if err != nil {
			fmt.Printf("Ошибка при получении матчей для сезона %d-%s: %v\n", leagueID, seasonName, err)
			continue
		}

		var missingMatches []models.Match
		for _, match := range matches {
			exists, err := db.IsMatchExists(match.ID)
			if err != nil {
				fmt.Printf("Ошибка проверки матча ID=%d: %v\n", match.ID, err)
				continue
			}
			if !exists {
				missingMatches = append(missingMatches, match)
			}
		}

		if len(missingMatches) > 0 {
			fmt.Printf("Найдено %d недостающих матчей для сезона %d-%s\n", len(missingMatches), leagueID, seasonName)
			processMatches(leagueID, seasonName, missingMatches)
		} else {
			fmt.Printf("Все матчи для сезона %d-%s уже есть в БД.\n", leagueID, seasonName)
		}
	}
}
