package db

import (
	"database/sql"
	"fmt"
	"football-data-miner/internal/models"
)

func SaveTeamIfNotExists(teamID int, teamName string) error {
	_, err := DB.Exec(`
        INSERT INTO teams (id, fullname)
        VALUES ($1, $2)
        ON CONFLICT (id) DO NOTHING
    `, teamID, teamName)
	return err
}

func SaveCoachIfNotExists(coachID int, coachName string) error {
	_, err := DB.Exec(`
        INSERT INTO coaches (id, fullname)
        VALUES ($1, $2)
        ON CONFLICT (id) DO NOTHING
    `, coachID, coachName)
	return err
}

func SavePlayerIfNotExists(playerID int, playerName string) error {
	_, err := DB.Exec(`
        INSERT INTO players (id, fullname)
        VALUES ($1, $2)
        ON CONFLICT (id) DO NOTHING
    `, playerID, playerName)
	return err
}
func SaveMatchDetails(match models.Match, leagueID int, season string, stats models.MatchStatistics, lineups []models.Lineup) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback()

	// Сохраняем матч
	if err := saveMatch(tx, match, leagueID, season); err != nil {
		return fmt.Errorf("ошибка сохранения матча ID=%d: %v", match.ID, err)
	}

	// Сохраняем статистику матча
	if !stats.IsDefault() {
		stats.MatchID = match.ID
		if err := saveMatchStatistics(tx, stats); err != nil {
			return fmt.Errorf("ошибка сохранения статистики матча ID=%d: %v", match.ID, err)
		}

		// Сохраняем составы игроков
		for _, lineup := range lineups {
			if lineup.IsEmpty() {
				continue // Пропускаем пустые составы
			}
			lineup.MatchID = match.ID
			if err := saveLineup(tx, lineup); err != nil {
				return fmt.Errorf("ошибка сохранения состава игрока ID=%d для матча ID=%d: %v", lineup.PlayerID, match.ID, err)
			}
		}
	} else {
		fmt.Printf("Матч ID=%d: статистика и составы отсутствуют. Пропускаем.\n", match.ID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %v", err)
	}

	fmt.Printf("Матч ID=%d успешно сохранен в БД.\n", match.ID)
	return nil
}

func saveMatch(tx *sql.Tx, match models.Match, leagueID int, season string) error {
	query := `
        INSERT INTO matches (
            id, date, league_id, season, home_team_id, away_team_id,
            home_score, away_score, home_coach_id, away_coach_id,
            home_formation, away_formation, round
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        ON CONFLICT (id) DO NOTHING
    `
	result, err := tx.Exec(query,
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
		match.Round,
	)
	if err != nil {
		return fmt.Errorf("ошибка сохранения матча ID=%d: %v", match.ID, err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("матч ID=%d уже существует", match.ID)
	}

	return nil
}
func saveMatchStatistics(tx *sql.Tx, stats models.MatchStatistics) error {
	if stats.IsDefault() {
		fmt.Printf("Матч ID=%d: статистика отсутствует. Пропускаем.\n", stats.MatchID)
		return nil
	}

	query := `
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
	        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33)
	        ON CONFLICT (match_id) DO NOTHING
	    `
	_, err := tx.Exec(query,
		stats.MatchID,
		stats.HomeBallPossession, stats.AwayBallPossession,
		stats.HomeShotsOnGoal, stats.AwayShotsOnGoal,
		stats.HomeShotsOffGoal, stats.AwayShotsOffGoal,
		stats.HomeTotalShots, stats.AwayTotalShots,
		stats.HomeBlockedShots, stats.AwayBlockedShots,
		stats.HomeShotsInsidebox, stats.AwayShotsInsidebox,
		stats.HomeShotsOutsidebox, stats.AwayShotsOutsidebox,
		stats.HomeFouls, stats.AwayFouls,
		stats.HomeCornerKicks, stats.AwayCornerKicks,
		stats.HomeOffsides, stats.AwayOffsides,
		stats.HomeYellowCards, stats.AwayYellowCards,
		stats.HomeRedCards, stats.AwayRedCards,
		stats.HomeGoalkeeperSaves, stats.AwayGoalkeeperSaves,
		stats.HomeTotalPasses, stats.AwayTotalPasses,
		stats.HomePassesAccurate, stats.AwayPassesAccurate,
		stats.HomePassesPercentage, stats.AwayPassesPercentage,
	)
	if err != nil {
		return fmt.Errorf("ошибка сохранения статистики: %v", err)
	}
	return nil
}

func saveLineup(tx *sql.Tx, lineup models.Lineup) error {
	query := `
        INSERT INTO lineups (
            match_id, team_id, player_id, pos, is_substitute,
            yellow_cards, red_cards, goals, assists,
            fouls_committed, fouls_drawn, dribbles_attempts,
            dribbles_success, duels_won, passes_total,
            passes_accuracy, tackles_total,tackles_blocks, tackles_interceptions, shots_total,
            shots_on, goals_conceded, goals_saved,
            minutes, captain, rating
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
            $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
        )
        ON CONFLICT (match_id, team_id, player_id) DO NOTHING
    `
	_, err := tx.Exec(query,
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
		lineup.TacklesBlocks,
		lineup.TacklesInterceptions,
		lineup.ShotsTotal,
		lineup.ShotsOn,
		lineup.GoalsConceded,
		lineup.GoalsSaved,
		lineup.Minutes,
		lineup.Captain,
		lineup.Rating,
	)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	return nil
}
