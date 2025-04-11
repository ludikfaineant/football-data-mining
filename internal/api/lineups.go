package api

import (
	"fmt"
	"football-data-miner/internal/db"
	"football-data-miner/internal/models"
	"strconv"
)

type LineupResponse struct {
	Response []struct {
		Coach struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"coach"`
		Formation string `json:"formation"`
		Team      struct {
			ID int `json:"id"`
		} `json:"team"`
		StartXI []struct {
			Player struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Pos  string `json:"pos"`
			} `json:"player"`
		} `json:"startXI"`
		Substitutes []struct {
			Player struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Pos  string `json:"pos"`
			} `json:"player"`
		} `json:"substitutes"`
	} `json:"response"`
}
type PlayersResponse struct {
	Response []struct {
		Team struct {
			ID int `json:"id"`
		} `json:"team"`
		Players []struct {
			Player struct {
				ID int `json:"id"`
			} `json:"player"`
			Statistics []PlayerStatistics `json:"statistics"`
		} `json:"players"`
	} `json:"response"`
}
type PlayerStatistics struct {
	Cards struct {
		Red    int `json:"red"`
		Yellow int `json:"yellow"`
	} `json:"cards"`
	Dribbles struct {
		Attempts int `json:"attempts"`
		Success  int `json:"success"`
	} `json:"dribbles"`
	Duels struct {
		Won   int `json:"won"`
		Total int `json:"total"`
	} `json:"duels"`
	Fouls struct {
		Committed int `json:"committed"`
		Drawn     int `json:"drawn"`
	} `json:"fouls"`
	Games struct {
		Captain    bool   `json:"captain"`
		Minutes    int    `json:"minutes"`
		Rating     string `json:"rating"`
		Substitute bool   `json:"substitute"`
	} `json:"games"`
	Goals struct {
		Assists  int `json:"assists"`
		Conceded int `json:"conceded"`
		Saves    int `json:"saves"`
		Total    int `json:"total"`
	} `json:"goals"`
	Passes struct {
		Accuracy string `json:"accuracy"`
		Total    int    `json:"total"`
	} `json:"passes"`
	Shots struct {
		On    int `json:"on"`
		Total int `json:"total"`
	} `json:"shots"`
	Tackles struct {
		Total int `json:"total"`
	} `json:"tackles"`
}

func MergeLineupAndPlayers(lineupResp LineupResponse, playersResp PlayersResponse, match *models.Match) []models.Lineup {
	var lineups []models.Lineup

	for _, teamLineup := range lineupResp.Response {
		err := db.SaveCoachIfNotExists(teamLineup.Coach.ID, teamLineup.Coach.Name)
		if err != nil {
			fmt.Printf("Ошибка сохранения тренера %d: %v\n", teamLineup.Coach.ID, err)
			continue
		}
		if teamLineup.Team.ID == match.HomeTeamID {
			match.HomeCoachID = teamLineup.Coach.ID
			match.HomeFormation = teamLineup.Formation
		} else {
			match.AwayCoachID = teamLineup.Coach.ID
			match.AwayFormation = teamLineup.Formation
		}
		processPlayers(match.ID, teamLineup.Team.ID, teamLineup.StartXI, playersResp, &lineups, false)
		processPlayers(match.ID, teamLineup.Team.ID, teamLineup.Substitutes, playersResp, &lineups, true)
	}

	return lineups
}

func processPlayers(matchID, teamID int, players []struct {
	Player struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Pos  string `json:"pos"`
	} `json:"player"`
}, playersResp PlayersResponse, lineups *[]models.Lineup, isSubstitute bool) {
	for _, player := range players {
		err := db.SavePlayerIfNotExists(player.Player.ID, player.Player.Name)
		if err != nil {
			fmt.Printf("Ошибка сохранения игрока %d: %v\n", player.Player.ID, err)
			continue
		}

		stats := findPlayerStats(teamID, player.Player.ID, playersResp)
		lineup := models.Lineup{
			MatchID:          matchID,
			PlayerID:         player.Player.ID,
			TeamID:           teamID,
			Position:         player.Player.Pos,
			IsSubstitute:     isSubstitute,
			YellowCards:      stats.Cards.Yellow,
			RedCards:         stats.Cards.Red,
			Goals:            safeInt(stats.Goals.Total),
			Assists:          safeInt(stats.Goals.Assists),
			FoulsCommitted:   safeInt(stats.Fouls.Committed),
			FoulsDrawn:       safeInt(stats.Fouls.Drawn),
			DribblesAttempts: safeInt(stats.Dribbles.Attempts),
			DribblesSuccess:  safeInt(stats.Dribbles.Success),
			DuelsWon:         safeInt(stats.Duels.Won),
			PassesTotal:      safeInt(stats.Passes.Total),
			PassesAccuracy:   parsePercentage(stats.Passes.Accuracy),
			TacklesTotal:     safeInt(stats.Tackles.Total),
			ShotsTotal:       safeInt(stats.Shots.Total),
			ShotsOn:          safeInt(stats.Shots.On),
			GoalsConceded:    safeInt(stats.Goals.Conceded),
			GoalsSaved:       safeInt(stats.Goals.Saves),
			Minutes:          safeInt(stats.Games.Minutes),
			Captain:          stats.Games.Captain,
			Rating:           parseRating(stats.Games.Rating),
		}
		*lineups = append(*lineups, lineup)
	}
}
func findPlayerStats(teamID, playerID int, playersResp PlayersResponse) *PlayerStatistics {
	for _, teamResp := range playersResp.Response {
		if teamResp.Team.ID == teamID {
			for _, playerData := range teamResp.Players {
				if playerData.Player.ID == playerID {
					return &playerData.Statistics[0]
				}
			}
		}
	}
	return &PlayerStatistics{}
}

func parseRating(rating string) float64 {
	if rating == "" {
		return 0.0
	}
	val, _ := strconv.ParseFloat(rating, 64)
	return val
}
