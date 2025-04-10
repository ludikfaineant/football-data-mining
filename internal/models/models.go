package models

type Team struct {
	ID       int    `json:"id"`
	Fullname string `json:"fullname"`
}

type Player struct {
	ID       int    `json:"id"`
	Fullname string `json:"fullname"`
}

type Coach struct {
	ID       int    `json:"id"`
	Fullname string `json:"fullname"`
}
type Match struct {
	ID            int    `json:"id"`
	Date          string `json:"date"`
	HomeTeamID    int    `json:"home_team_id"`
	AwayTeamID    int    `json:"away_team_id"`
	HomeTeamName  string `json:"home_team_name"`
	AwayTeamName  string `json:"away_team_name"`
	HomeScore     int    `json:"home_score"`
	AwayScore     int    `json:"away_score"`
	HomeCoachID   int    `json:"home_coach_id"`
	AwayCoachID   int    `json:"away_coach_id"`
	HomeFormation string `json:"home_formation"`
	AwayFormation string `json:"away_formation"`
}

type MatchStatistics struct {
	MatchID              int `json:"match_id"`
	HomeBallPossession   int `json:"home_ball_possession"`
	AwayBallPossession   int `json:"away_ball_possession"`
	HomeShotsOnGoal      int `json:"home_shots_on_goal"`
	AwayShotsOnGoal      int `json:"away_shots_on_goal"`
	HomeShotsOffGoal     int `json:"home_shots_off_goal"`
	AwayShotsOffGoal     int `json:"away_shots_off_goal"`
	HomeTotalShots       int `json:"home_total_shots"`
	AwayTotalShots       int `json:"away_total_shots"`
	HomeBlockedShots     int `json:"home_blocked_shots"`
	AwayBlockedShots     int `json:"away_blocked_shots"`
	HomeShotsInsidebox   int `json:"home_shots_insidebox"`
	AwayShotsInsidebox   int `json:"away_shots_insidebox"`
	HomeShotsOutsidebox  int `json:"home_shots_outsidebox"`
	AwayShotsOutsidebox  int `json:"away_shots_outsidebox"`
	HomeFouls            int `json:"home_fouls"`
	AwayFouls            int `json:"away_fouls"`
	HomeCornerKicks      int `json:"home_corner_kicks"`
	AwayCornerKicks      int `json:"away_corner_kicks"`
	HomeOffsides         int `json:"home_offsides"`
	AwayOffsides         int `json:"away_offsides"`
	HomeYellowCards      int `json:"home_yellow_cards"`
	AwayYellowCards      int `json:"away_yellow_cards"`
	HomeRedCards         int `json:"home_red_cards"`
	AwayRedCards         int `json:"away_red_cards"`
	HomeGoalkeeperSaves  int `json:"home_goalkeeper_saves"`
	AwayGoalkeeperSaves  int `json:"away_goalkeeper_saves"`
	HomeTotalPasses      int `json:"home_total_passes"`
	AwayTotalPasses      int `json:"away_total_passes"`
	HomePassesAccurate   int `json:"home_passes_accurate"`
	AwayPassesAccurate   int `json:"away_passes_accurate"`
	HomePassesPercentage int `json:"home_passes_percentage"`
	AwayPassesPercentage int `json:"away_passes_percentage"`
}

type Lineup struct {
	MatchID          int     `json:"match_id"`
	TeamID           int     `json:"team_id"`
	PlayerID         int     `json:"player_id"`
	Position         string  `json:"pos"`
	IsSubstitute     bool    `json:"is_substitute"`
	YellowCards      int     `json:"yellow_cards"`
	RedCards         int     `json:"red_cards"`
	Goals            int     `json:"goals"`
	Assists          int     `json:"assists"`
	FoulsCommitted   int     `json:"fouls_committed"`
	FoulsDrawn       int     `json:"fouls_drawn"`
	DribblesAttempts int     `json:"dribbles_attempts"`
	DribblesSuccess  int     `json:"dribbles_success"`
	DuelsWon         int     `json:"duels_won"`
	PassesTotal      int     `json:"passes_total"`
	PassesAccuracy   int     `json:"passes_accuracy"`
	TacklesTotal     int     `json:"tackles_total"`
	ShotsTotal       int     `json:"shots_total"`
	ShotsOn          int     `json:"shots_on"`
	GoalsConceded    int     `json:"goals_conceded"`
	GoalsSaved       int     `json:"goals_saved"`
	Minutes          int     `json:"minutes"`
	Captain          bool    `json:"captain"`
	Rating           float64 `json:"rating"`
}

type MatchesOfSeason struct {
	Response []struct {
		Fixture struct {
			ID   int    `json:"id"`
			Date string `json:"date"`
		} `json:"fixture"`
		Teams struct {
			Home struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"home"`
			Away struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"away"`
		} `json:"teams"`
		Goals struct {
			Home interface{} `json:"home"`
			Away interface{} `json:"away"`
		} `json:"goals"`
	} `json:"response"`
}
