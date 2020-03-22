package data

import (
	"log"
	"math"

	"github.com/kcapp/api/models"
)

// AddVisit will write thegiven visit to database
func AddVisit(visit models.Visit) (*models.Visit, error) {
	currentScore, err := GetPlayerScore(visit.PlayerID, visit.LegID)
	if err != nil {
		return nil, err
	}

	// TODO Don't allow to save score for same player twice in a row
	// Only allow saving score for leg.current_player_id ?

	leg, err := GetLeg(visit.LegID)
	if err != nil {
		return nil, err
	}

	// TODO Don't allow to insert if leg is finished?

	match, err := GetMatch(leg.MatchID)
	if err != nil {
		return nil, err
	}

	if match.MatchType.ID == models.X01 || match.MatchType.ID == models.X01HANDICAP {
		// Only set busts for x01 match modes
		visit.SetIsBust(currentScore)
	}

	// Determine who the next player will be
	players, err := GetPlayersScore(visit.LegID)
	if err != nil {
		return nil, err
	}

	currentPlayerOrder := 1
	order := make(map[int]int)
	for _, player := range players {
		if player.PlayerID == visit.PlayerID {
			currentPlayerOrder = player.Order
		}
		order[player.Order] = player.PlayerID
	}
	nextPlayerID := order[(currentPlayerOrder%len(players))+1]

	tx, err := models.DB.Begin()
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(`
		INSERT INTO score(
			leg_id, player_id,
			first_dart, first_dart_multiplier,
			second_dart, second_dart_multiplier,
			third_dart, third_dart_multiplier,
			is_bust, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())`,
		visit.LegID, visit.PlayerID,
		visit.FirstDart.Value, visit.FirstDart.Multiplier,
		visit.SecondDart.Value, visit.SecondDart.Multiplier,
		visit.ThirdDart.Value, visit.ThirdDart.Multiplier,
		visit.IsBust)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	_, err = tx.Exec(`UPDATE leg SET current_player_id = ?, updated_at = NOW() WHERE id = ?`, nextPlayerID, visit.LegID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()

	log.Printf("[%d] Added score for player %d, (%d-%d, %d-%d, %d-%d, %t)", visit.LegID, visit.PlayerID, visit.FirstDart.Value.Int64,
		visit.FirstDart.Multiplier, visit.SecondDart.Value.Int64, visit.SecondDart.Multiplier, visit.ThirdDart.Value.Int64, visit.ThirdDart.Multiplier,
		visit.IsBust)

	if match.MatchType.ID == models.SHOOTOUT {
		if ((len(leg.Visits)+1)*3)%(9*len(leg.Players)) == 0 {
			err = FinishLegNew(visit)
			if err != nil {
				return nil, err
			}
		}
	} else if match.MatchType.ID == models.DARTSATX {
		if ((len(leg.Visits)+1)*3)%(99*len(leg.Players)) == 0 {
			err = FinishLegNew(visit)
			if err != nil {
				return nil, err
			}
		}
	} else if match.MatchType.ID == models.CRICKET {
		players, err := GetLegPlayers(visit.LegID)
		if err != nil {
			return nil, err
		}

		// Did current player close all numbers?
		closedPlayers := make(map[int]*models.Player2Leg, 0)
		for _, player := range players {
			if player.PlayerID != visit.PlayerID {
				continue
			}
			closed := true
			for _, dart := range []int{15, 16, 17, 18, 19, 20, 25} {
				if player.Hits[dart] == nil || player.Hits[dart].Total < 3 {
					closed = false
					break
				}
			}
			if closed {
				closedPlayers[player.PlayerID] = player
			}
		}

		// What is the lowest score?
		lowestScore := math.MaxInt32
		for _, player := range players {
			if player.CurrentScore < lowestScore {
				lowestScore = player.CurrentScore
			}
		}

		// If current player closed all numbers and has the lowest score, it's finished
		if player, ok := closedPlayers[visit.PlayerID]; ok {
			if player.CurrentScore == lowestScore {
				err = FinishLegNew(visit)
				if err != nil {
					return nil, err
				}
			}
		}
	} else {
		if !visit.IsBust && visit.IsCheckout(currentScore) {
			// Finalize leg, since leg is finished!
			err = FinishLegNew(visit)
			if err != nil {
				return nil, err
			}
		}
	}
	return &visit, nil
}

// ModifyVisit modify the scores of a visit
func ModifyVisit(visit models.Visit) error {
	// FIXME: We need to check if this is a checkout/bust
	stmt, err := models.DB.Prepare(`
		UPDATE score SET
    		first_dart = ?,
    		first_dart_multiplier = ?,
    		second_dart = ?,
    		second_dart_multiplier = ?,
    		third_dart = ?,
		    third_dart_multiplier = ?,
			updated_at = NOW()
		WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(visit.FirstDart.Value, visit.FirstDart.Multiplier, visit.SecondDart.Value, visit.SecondDart.Multiplier,
		visit.ThirdDart.Value, visit.ThirdDart.Multiplier, visit.ID)
	if err != nil {
		return err
	}
	log.Printf("[%d] Modified score %d, throws: (%d-%d, %d-%d, %d-%d)", visit.LegID, visit.ID, visit.FirstDart.Value.Int64,
		visit.FirstDart.Multiplier, visit.SecondDart.Value.Int64, visit.SecondDart.Multiplier, visit.ThirdDart.Value.Int64, visit.ThirdDart.Multiplier)

	return nil
}

// DeleteVisit will delete the visit for the given ID
func DeleteVisit(id int) error {
	visit, err := GetVisit(id)
	if err != nil {
		return err
	}
	tx, err := models.DB.Begin()
	if err != nil {
		return err
	}
	// Delete the visit
	_, err = tx.Exec("DELETE FROM score WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Set current player to the player of the last visit
	_, err = tx.Exec("UPDATE leg SET current_player_id = ? WHERE id = ?", visit.PlayerID, visit.LegID)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	log.Printf("[%d] Deleted visit %d", visit.LegID, visit.ID)
	return nil
}

// DeleteLastVisit will delete the last visit for the given leg
func DeleteLastVisit(legID int) error {
	visits, err := GetLegVisits(legID)
	if err != nil {
		return err
	}

	if len(visits) > 0 {
		err := DeleteVisit(visits[len(visits)-1].ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPlayerVisits will return all visits for a given player
func GetPlayerVisits(id int) ([]*models.Visit, error) {
	rows, err := models.DB.Query(`
		SELECT
			id, leg_id, player_id,
			first_dart, first_dart_multiplier,
			second_dart, second_dart_multiplier,
			third_dart, third_dart_multiplier,
			is_bust,
			created_at,
			updated_at
		FROM score s
		WHERE player_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	visits := make([]*models.Visit, 0)
	for rows.Next() {
		v := new(models.Visit)
		v.FirstDart = new(models.Dart)
		v.SecondDart = new(models.Dart)
		v.ThirdDart = new(models.Dart)
		err := rows.Scan(&v.ID, &v.LegID, &v.PlayerID,
			&v.FirstDart.Value, &v.FirstDart.Multiplier,
			&v.SecondDart.Value, &v.SecondDart.Multiplier,
			&v.ThirdDart.Value, &v.ThirdDart.Multiplier,
			&v.IsBust, &v.CreatedAt, &v.UpdatedAt)
		if err != nil {
			return nil, err
		}
		visits = append(visits, v)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return visits, nil
}

// GetLegVisits will return all visits for a given leg
func GetLegVisits(id int) ([]*models.Visit, error) {
	rows, err := models.DB.Query(`
		SELECT
			id, leg_id, player_id,
			first_dart, first_dart_multiplier,
			second_dart, second_dart_multiplier,
			third_dart, third_dart_multiplier,
			is_bust,
			created_at,
			updated_at
		FROM score s
		WHERE leg_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	visits := make([]*models.Visit, 0)
	for rows.Next() {
		v := new(models.Visit)
		v.FirstDart = new(models.Dart)
		v.SecondDart = new(models.Dart)
		v.ThirdDart = new(models.Dart)
		err := rows.Scan(&v.ID, &v.LegID, &v.PlayerID,
			&v.FirstDart.Value, &v.FirstDart.Multiplier,
			&v.SecondDart.Value, &v.SecondDart.Multiplier,
			&v.ThirdDart.Value, &v.ThirdDart.Multiplier,
			&v.IsBust, &v.CreatedAt, &v.UpdatedAt)
		if err != nil {
			return nil, err
		}
		visits = append(visits, v)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return visits, nil
}

// GetVisit will return the visit with the given ID
func GetVisit(id int) (*models.Visit, error) {
	v := new(models.Visit)
	v.FirstDart = new(models.Dart)
	v.SecondDart = new(models.Dart)
	v.ThirdDart = new(models.Dart)
	err := models.DB.QueryRow(`
		SELECT
			id, leg_id, player_id,
			first_dart, first_dart_multiplier,
			second_dart, second_dart_multiplier,
			third_dart, third_dart_multiplier,
			is_bust,
			created_at,
			updated_at
		FROM score s
		WHERE s.id = ?`, id).Scan(&v.ID, &v.LegID, &v.PlayerID,
		&v.FirstDart.Value, &v.FirstDart.Multiplier,
		&v.SecondDart.Value, &v.SecondDart.Multiplier,
		&v.ThirdDart.Value, &v.ThirdDart.Multiplier,
		&v.IsBust, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// GetLastVisits will return the last N visit for the given leg
func GetLastVisits(legID int, num int) (map[int]*models.Visit, error) {
	rows, err := models.DB.Query(`
			SELECT
				player_id,
				first_dart, first_dart_multiplier,
				second_dart, second_dart_multiplier,
				third_dart, third_dart_multiplier
			FROM score
			WHERE leg_id = ? AND is_bust = 0
			ORDER BY id DESC LIMIT ?`, legID, num)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	visits := make(map[int]*models.Visit)
	for rows.Next() {
		v := new(models.Visit)
		v.FirstDart = new(models.Dart)
		v.SecondDart = new(models.Dart)
		v.ThirdDart = new(models.Dart)
		err := rows.Scan(&v.PlayerID,
			&v.FirstDart.Value, &v.FirstDart.Multiplier,
			&v.SecondDart.Value, &v.SecondDart.Multiplier,
			&v.ThirdDart.Value, &v.ThirdDart.Multiplier)
		if err != nil {
			return nil, err
		}
		visits[v.PlayerID] = v
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return visits, nil
}

// GetPlayerVisitCount will return a count of each visit for a given player
func GetPlayerVisitCount(playerID int) ([]*models.Visit, error) {
	rows, err := models.DB.Query(`
		SELECT
			player_id,
			first_dart, first_dart_multiplier,
			second_dart, second_dart_multiplier,
			third_dart, third_dart_multiplier,
			COUNT(*) AS 'visits'
		FROM score s
			WHERE player_id = ?
		GROUP BY
			player_id, first_dart, first_dart_multiplier,
			second_dart, second_dart_multiplier,
			third_dart, third_dart_multiplier`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]*models.Visit)
	for rows.Next() {
		v := new(models.Visit)
		v.FirstDart = new(models.Dart)
		v.SecondDart = new(models.Dart)
		v.ThirdDart = new(models.Dart)
		err := rows.Scan(&v.PlayerID,
			&v.FirstDart.Value, &v.FirstDart.Multiplier,
			&v.SecondDart.Value, &v.SecondDart.Multiplier,
			&v.ThirdDart.Value, &v.ThirdDart.Multiplier,
			&v.Count)
		if err != nil {
			return nil, err
		}

		s := v.GetVisitString()
		if val, ok := m[s]; ok {
			val.Count += v.Count
		} else {
			m[s] = v
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	visits := make([]*models.Visit, 0)
	for _, v := range m {
		visits = append(visits, v)
	}

	return visits, nil
}

// GetRandomLegForPlayer will return a random leg for a given player and starting score
func GetRandomLegForPlayer(playerID int, startingScore int) ([]*models.Visit, error) {
	var legID int
	err := models.DB.QueryRow(`
		SELECT
			l.id
		FROM leg l
			JOIN player2leg p2l ON p2l.leg_id = l.id
		WHERE l.is_finished = 1 AND l.winner_id = ? AND l.starting_score = ? AND l.has_scores = 1
		GROUP BY l.id
			HAVING COUNT(DISTINCT p2l.player_id) = 2
		ORDER BY RAND()
		LIMIT 1`, playerID, startingScore).Scan(&legID)
	if err != nil {
		return nil, err
	}

	rows, err := models.DB.Query(`
		SELECT
			id, leg_id, player_id,
			first_dart, first_dart_multiplier,
			second_dart, second_dart_multiplier,
			third_dart, third_dart_multiplier,
			is_bust,
			created_at,
			updated_at
		FROM score s
		WHERE leg_id = ? AND player_id = ?`, legID, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	visits := make([]*models.Visit, 0)
	for rows.Next() {
		v := new(models.Visit)
		v.FirstDart = new(models.Dart)
		v.SecondDart = new(models.Dart)
		v.ThirdDart = new(models.Dart)
		err := rows.Scan(&v.ID, &v.LegID, &v.PlayerID,
			&v.FirstDart.Value, &v.FirstDart.Multiplier,
			&v.SecondDart.Value, &v.SecondDart.Multiplier,
			&v.ThirdDart.Value, &v.ThirdDart.Multiplier,
			&v.IsBust, &v.CreatedAt, &v.UpdatedAt)
		if err != nil {
			return nil, err
		}
		visits = append(visits, v)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return visits, nil
}

// GetDartStatistics will return statistics of times hit for a given dart
func GetDartStatistics(dart int) (map[int]*models.Hits, error) {
	rows, err := models.DB.Query(`
		SELECT player_id, singles, doubles, triples
		FROM (
			SELECT s.player_id,
				SUM(IF(s.first_dart = ? AND s.first_dart_multiplier = 1, 1, 0) +
					IF(s.second_dart = ? AND s.second_dart_multiplier = 1, 1, 0) +
					IF(s.third_dart = ? AND s.third_dart_multiplier = 1, 1, 0)) AS 'singles',
				SUM(IF(s.first_dart = ? AND s.first_dart_multiplier = 2, 1, 0) +
					IF(s.second_dart = ? AND s.second_dart_multiplier = 2, 1, 0) +
					IF(s.third_dart = ? AND s.third_dart_multiplier = 2, 1, 0)) AS 'doubles',
				SUM(IF(s.first_dart = ? AND s.first_dart_multiplier = 3, 1, 0) +
					IF(s.second_dart = ? AND s.second_dart_multiplier = 3, 1, 0) +
					IF(s.third_dart = ? AND s.third_dart_multiplier = 3, 1, 0)) AS 'triples'
			FROM score s
			JOIN leg l ON l.id = s.leg_id
			JOIN matches m ON m.id = l.match_id
			WHERE s.is_bust = 0
			GROUP BY player_id
		) scores`, dart, dart, dart, dart, dart, dart, dart, dart, dart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int]*models.Hits)
	for rows.Next() {
		h := new(models.Hits)
		var playerID int
		err := rows.Scan(&playerID, &h.Singles, &h.Doubles, &h.Triples)
		if err != nil {
			return nil, err
		}
		m[playerID] = h
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

// calculateCricketScore will calculate the score for each player for the given dart
func calculateCricketScore(playerID int, dart *models.Dart, scores map[int]*models.Player2Leg) {
	if !dart.Value.Valid {
		return
	}

	score := int(dart.Value.Int64)
	darts := []int{15, 16, 17, 18, 19, 20, 25}
	found := false
	for _, val := range darts {
		if val == score {
			found = true
		}
	}
	if !found {
		return
	}

	hitsMap := scores[playerID].Hits
	if _, ok := hitsMap[score]; !ok {
		hitsMap[score] = new(models.Hits)
	}
	hits := hitsMap[score].Total
	hitsMap[score].Total += int(dart.Multiplier)
	multiplier := hitsMap[score].Total - hits
	if hits < 3 {
		multiplier = hitsMap[score].Total - 3
	}
	points := int(dart.Value.Int64) * multiplier

	if hitsMap[score].Total > 3 {
		for id, p2l := range scores {
			if id == playerID {
				continue
			}
			if val, ok := p2l.Hits[score]; ok {
				if val.Total < 3 {
					p2l.CurrentScore += points
				}
			} else {
				p2l.CurrentScore += points
			}
		}
	}
}
