package api

import (
	"fmt"
	"football-data-miner/internal/cache"
	"football-data-miner/internal/models"
	"strconv"
	"strings"
)

type TeamStatistics struct {
	Team struct {
		ID int `json:"id"`
	} `json:"team"`
	Statistics []struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	} `json:"statistics"`
}

func ParseStatistics(fixtureID int, teamStats []TeamStatistics) (models.MatchStatistics, error) {
	match, err := cache.GetMatchFromRedis(fixtureID)
	if err != nil {
		return models.MatchStatistics{}, err
	}

	stats := models.MatchStatistics{
		MatchID: fixtureID,
	}

	for _, teamStat := range teamStats {
		isHome := teamStat.Team.ID == match.HomeTeamID

		for _, s := range teamStat.Statistics {
			switch s.Type {
			case "Ball Possession":
				possession := parsePercentage(s.Value)
				if isHome {
					stats.HomeBallPossession = possession
				} else {
					stats.AwayBallPossession = possession
				}
			case "Shots on Goal":
				val := safeInt(s.Value) // <- Исправление
				if isHome {
					stats.HomeShotsOnGoal = val
				} else {
					stats.AwayShotsOnGoal = val
				}
			case "Shots off Goal":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeShotsOffGoal = val
				} else {
					stats.AwayShotsOffGoal = val
				}
			case "Total Shots":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeTotalShots = val
				} else {
					stats.AwayTotalShots = val
				}
			case "Blocked Shots":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeBlockedShots = val
				} else {
					stats.AwayBlockedShots = val
				}
			case "Shots insidebox":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeShotsInsidebox = val
				} else {
					stats.AwayShotsInsidebox = val
				}
			case "Shots outsidebox":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeShotsOutsidebox = val
				} else {
					stats.AwayShotsOutsidebox = val
				}
			case "Fouls":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeFouls = val
				} else {
					stats.AwayFouls = val
				}
			case "Corner Kicks":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeCornerKicks = val
				} else {
					stats.AwayCornerKicks = val
				}
			case "Offsides":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeOffsides = val
				} else {
					stats.AwayOffsides = val
				}
			case "Yellow Cards":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeYellowCards = val
				} else {
					stats.AwayYellowCards = val
				}
			case "Red Cards":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeRedCards = val
				} else {
					stats.AwayRedCards = val
				}
			case "Goalkeeper Saves":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeGoalkeeperSaves = val
				} else {
					stats.AwayGoalkeeperSaves = val
				}
			case "Total passes":
				val := safeInt(s.Value)
				if isHome {
					stats.HomeTotalPasses = val
				} else {
					stats.AwayTotalPasses = val
				}
			case "Passes accurate":
				val := safeInt(s.Value)
				if isHome {
					stats.HomePassesAccurate = val
				} else {
					stats.AwayPassesAccurate = val
				}
			case "Passes %":
				passesPct := parsePercentage(s.Value)
				if isHome {
					stats.HomePassesPercentage = passesPct
				} else {
					stats.AwayPassesPercentage = passesPct
				}
			}
		}
	}

	return stats, nil
}

func parsePercentage(value interface{}) int {
	if value == nil {
		return 0
	}
	strValue := fmt.Sprintf("%v", value)
	strValue = strings.TrimSuffix(strValue, "%")
	num, _ := strconv.Atoi(strValue)
	return num
}
