package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"football-data-miner/internal/models"
	"time"

	// Импортируем пакет api

	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client

func InitRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		panic(err)
	}
}

func IsCacheEmpty() bool {
	keys, _ := rdb.Keys(context.Background(), "matches:season:*").Result()
	return len(keys) == 0
}

func GetSeasonMatches(leagueID int, season string) ([]models.Match, error) {
	key := getSeasonKey(leagueID, season)
	matchesJSON, err := rdb.Get(context.Background(), key).Result()
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении матчей: %v", err)
	}

	var matches []models.Match // Используем models.Match
	err = json.Unmarshal([]byte(matchesJSON), &matches)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга матчей: %v", err)
	}

	return matches, nil
}

func CacheSeasonMatches(leagueID int, season string, matches []models.Match) error { // Используем models.Match
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := getSeasonKey(leagueID, season)
	matchesJSON, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %v", err)
	}

	err = rdb.Set(context.Background(), key, string(matchesJSON), 7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("ошибка сохранения в Redis: %v", err)
	}

	return nil
}
func MarkMatchAsProcessed(leagueID int, season string, matchID int) {
	processedKey := fmt.Sprintf("processed_matches:season:%d:%s", leagueID, season)

	err := rdb.SAdd(context.Background(), processedKey, matchID).Err()
	if err != nil {
		fmt.Printf("Ошибка при добавлении матча ID=%d в множество обработанных: %v\n", matchID, err)
		return
	}

	fmt.Printf("Матч ID=%d помечен как обработанный\n", matchID)

	err = rdb.Expire(context.Background(), processedKey, 7*24*time.Hour).Err()
	if err != nil {
		fmt.Printf("Ошибка при установке TTL для ключа %s: %v\n", processedKey, err)
	}
}
func IsMatchProcessed(leagueID int, season string, matchID int) (bool, error) {
	processedKey := getProcessedKey(leagueID, season)
	isProcessed, err := rdb.SIsMember(context.Background(), processedKey, matchID).Result()
	return isProcessed, err // Теперь возвращаем два значения
}

func GetAllSeasonKeys() ([]string, error) {
	return rdb.Keys(context.Background(), "matches:season:*").Result()
}

func GetMatchFromRedis(fixtureID int) (*models.Match, error) {
	ctx := context.Background()
	keys, err := rdb.Keys(context.Background(), "matches:season:*").Result()
	if err != nil {
		return nil, fmt.Errorf("ошибка при поиске матчей: %v", err)
	}

	for _, key := range keys {
		pipe := rdb.Pipeline()
		getCmd := pipe.Get(ctx, key)
		pipe.Exec(ctx)
		result, err := getCmd.Result()
		if err != nil {
			continue
		}

		var matches []models.Match
		err = json.Unmarshal([]byte(result), &matches)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if match.ID == fixtureID {
				return &match, nil
			}
		}
	}

	return nil, fmt.Errorf("матч ID=%d не найден", fixtureID)
}

func getSeasonKey(leagueID int, season string) string {
	return fmt.Sprintf("matches:season:%d:%s", leagueID, season)
}

func getProcessedKey(leagueID int, season string) string {
	return fmt.Sprintf("processed_matches:season:%d:%s", leagueID, season)
}
