package db

import (
	"fmt"
	"football-data-miner/internal/models"
)

func SaveTeamIfNotExists(teamID int, teamName string) error {
	_, err := dbConn.Exec(`
        INSERT INTO teams (id, fullname)
        VALUES ($1, $2)
        ON CONFLICT (id) DO NOTHING
    `, teamID, teamName)
	return err
}

func SaveCoachIfNotExists(coachID int, coachName string) error {
	_, err := dbConn.Exec(`
        INSERT INTO coaches (id, fullname)
        VALUES ($1, $2)
        ON CONFLICT (id) DO NOTHING
    `, coachID, coachName)
	return err
}

func SavePlayerIfNotExists(playerID int, playerName string) error {
	_, err := dbConn.Exec(`
        INSERT INTO players (id, fullname)
        VALUES ($1, $2)
        ON CONFLICT (id) DO NOTHING
    `, playerID, playerName)
	return err
}
func SaveMatchDetails(match models.Match, leagueID int, season string, stats models.MatchStatistics, lineups []models.Lineup) error {
	tx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
        INSERT INTO matches (
            id, date, league_id, season, home_team_id, away_team_id,
            home_score, away_score, home_coach_id, away_coach_id,
            home_formation, away_formation
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT (id) DO NOTHING
    `,
		match.ID,
		match.Date,
		leagueID,
		season,
		match.HomeTeamID,
		match.AwayTeamID,
		match.HomeScore,
		match.AwayScore,
		match.HomeCoachID,
		match.AwayCoachID,
		match.HomeFormation,
		match.AwayFormation,
	)
	if err != nil {
		return fmt.Errorf("ошибка сохранения матча: %v", err)
	}

	// 2. Сохраняем статистику матча
	_, err = tx.Exec(`
        INSERT INTO match_statistics (
            match_id, home_ball_possession, away_ball_possession,
            home_shots_on_goal, away_shots_on_goal,
            home_shots_off_goal, away_shots_off_goal,
            home_total_shots, away_total_shots,
            home_blocked_shots, away_blocked_shots,
            home_shots_insidebox, away_shots_insidebox,
            home_shots_outsidebox, away_shots_outsidebox,
            home_fouls, away_fouls,
            home_corner_kicks, away_corner_kicks,
            home_offsides, away_offsides,
            home_yellow_cards, away_yellow_cards,
            home_red_cards, away_red_cards,
            home_goalkeeper_saves, away_goalkeeper_saves,
            home_total_passes, away_total_passes,
            home_passes_accurate, away_passes_accurate,
            home_passes_percentage, away_passes_percentage
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
        )
        ON CONFLICT (match_id) DO NOTHING
    `,
		stats.MatchID,
		stats.HomeBallPossession,
		stats.AwayBallPossession,
		stats.HomeShotsOnGoal,
		stats.AwayShotsOnGoal,
		stats.HomeShotsOffGoal,
		stats.AwayShotsOffGoal,
		stats.HomeTotalShots,
		stats.AwayTotalShots,
		stats.HomeBlockedShots,
		stats.AwayBlockedShots,
		stats.HomeShotsInsidebox,
		stats.AwayShotsInsidebox,
		stats.HomeShotsOutsidebox,
		stats.AwayShotsOutsidebox,
		stats.HomeFouls,
		stats.AwayFouls,
		stats.HomeCornerKicks,
		stats.AwayCornerKicks,
		stats.HomeOffsides,
		stats.AwayOffsides,
		stats.HomeYellowCards,
		stats.AwayYellowCards,
		stats.HomeRedCards,
		stats.AwayRedCards,
		stats.HomeGoalkeeperSaves,
		stats.AwayGoalkeeperSaves,
		stats.HomeTotalPasses,
		stats.AwayTotalPasses,
		stats.HomePassesAccurate,
		stats.AwayPassesAccurate,
		stats.HomePassesPercentage,
		stats.AwayPassesPercentage,
	)
	if err != nil {
		return err
	}

	// 3. Сохраняем составы игроков
	for _, lineup := range lineups {
		_, err = tx.Exec(`
            INSERT INTO lineups (
                match_id, team_id, player_id, pos, is_substitute,
                yellow_cards, red_cards, goals, assists,
                fouls_committed, fouls_drawn, dribbles_attempts,
                dribbles_success, duels_won, passes_total,
                passes_accuracy, tackles_total, shots_total,
                shots_on, goals_conceded, goals_saved,
                minutes, captain, rating
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
                $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
            )
            ON CONFLICT (match_id, team_id, player_id) DO NOTHING
        `,
			lineup.MatchID,
			lineup.TeamID,
			lineup.PlayerID,
			lineup.Position,
			lineup.IsSubstitute,
			lineup.YellowCards,
			lineup.RedCards,
			lineup.Goals,
			lineup.Assists,
			lineup.FoulsCommitted,
			lineup.FoulsDrawn,
			lineup.DribblesAttempts,
			lineup.DribblesSuccess,
			lineup.DuelsWon,
			lineup.PassesTotal,
			lineup.PassesAccuracy,
			lineup.TacklesTotal,
			lineup.ShotsTotal,
			lineup.ShotsOn,
			lineup.GoalsConceded,
			lineup.GoalsSaved,
			lineup.Minutes,
			lineup.Captain,
			lineup.Rating,
		)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка коммита: %v", err)
	}

	tx = nil
	return nil
}
