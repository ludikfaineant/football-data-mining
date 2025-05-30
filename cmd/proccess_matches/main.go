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
		if match.HomeScore == nil {
			fmt.Print("Shtoto ne tak")
			return true
		}

		if match.Date == "" || (match.Date < "2025-05-21 05:05:00" && leagueID != 94 && leagueID != 144) || match.Date < "2025-05-11 05:05:00" {
			cache.MarkMatchAsProcessed(leagueID, season, match.ID)
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

		parsedStats, _ := api.ParseStatistics(match.ID, stats)
		parsedLineups := api.MergeLineupAndPlayers(lineups, players, &match)
		db.SaveMatchDetails(match, leagueID, season, parsedStats, parsedLineups)
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
	missingMatches, err := db.GetMissingMatchesFromDB(db.DB)
	if err != nil {
		fmt.Printf("Ошибка получения недостающих матчей: %v\n", err)
		return
	}

	if len(missingMatches) == 0 {
		fmt.Println("Недостающих матчей не найдено.")
		return
	}

	fmt.Printf("Найдено %d недостающих матчей для обработки.\n", len(missingMatches))

	for _, match := range missingMatches {
		leagueID, season, err := db.GetLeagueAndSeasonForMatch(match.ID)
		if err != nil {
			fmt.Printf("Ошибка получения лиги и сезона для матча ID=%d: %v\n", match.ID, err)
			continue
		}

		fmt.Printf("Обрабатываем матч ID=%d (лига %d, сезон %s)\n", match.ID, leagueID, season)

		stats, canContinue, err := api.FetchStatistics(match.ID)
		if err != nil {
			fmt.Printf("Ошибка статистики для матча ID=%d: %v\n", match.ID, err)
			continue
		}

		lineups, err := api.FetchLineups(match.ID)
		if err != nil {
			fmt.Printf("Ошибка составов для матча ID=%d: %v\n", match.ID, err)
			continue
		}

		players, err := api.FetchPlayers(match.ID)
		if err != nil {
			fmt.Printf("Ошибка событий для матча ID=%d: %v\n", match.ID, err)
			continue
		}

		parsedStats, _ := api.ParseStatistics(match.ID, stats)
		parsedLineups := api.MergeLineupAndPlayers(lineups, players, &match)
		db.SaveMatchDetails(match, leagueID, season, parsedStats, parsedLineups)

		cache.MarkMatchAsProcessed(leagueID, season, match.ID)
		if !canContinue {
			fmt.Printf("Прерываем обработку после матча ID=%d\n", match.ID)
			break
		}
	}
}

func MarkProcessedMatchesInRedis(leagueID int, season string) error {
	fmt.Println("Получение списка обработанных матчей из БД...")
	processedMatches, err := db.GetProcessedMatches(leagueID, season)
	if err != nil {
		return fmt.Errorf("ошибка получения обработанных матчей из БД: %v", err)
	}

	fmt.Printf("Найдено %d обработанных матчей для кэширования в Redis.\n", len(processedMatches))
	for _, match := range processedMatches {
		cache.MarkMatchAsProcessed(leagueID, season, match)
	}

	fmt.Println("Все обработанные матчи успешно сохранены в Redis.")
	return nil
}
