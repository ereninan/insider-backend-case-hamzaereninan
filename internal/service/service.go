package service

import (
	"database/sql"
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/insider/football-league/internal/models"
)

type LeagueManager interface {
	PlayNextWeek() ([]models.Match, error)
	PlayAll() ([]models.Match, error)
	GetStandings() ([]models.Team, error)
	GetPredictions() ([]models.Prediction, error)
	GetAllMatches() (map[int][]models.Match, error)
	GetWeekSummary(week int) (*models.WeekSummary, error)
	EditMatchResult(matchID, homeScore, awayScore int) error
}

type leagueService struct {
	db *sql.DB
}

func NewLeagueManager(db *sql.DB) (LeagueManager, error) {
	svc := &leagueService{
		db: db,
	}
	
	err := svc.createFixtures()
	if err != nil {
		return nil, err
	}
	
	return svc, nil
}

func (s *leagueService) GetStandings() ([]models.Team, error) {
	rows, err := s.db.Query(`
		SELECT id, name, strength, played, won, drawn, lost, gf, ga, gd, points
		FROM teams
		ORDER BY points DESC, gd DESC, gf DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		err := rows.Scan(
			&t.ID, &t.Name, &t.Strength,
			&t.Played, &t.Won, &t.Drawn, &t.Lost,
			&t.GF, &t.GA, &t.GD, &t.Points,
		)
		if err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func (s *leagueService) PlayNextWeek() ([]models.Match, error) {
	week, err := s.getCurrentWeek()
	if err != nil {
		return nil, err
	}
	if week == 0 {
		return nil, errors.New("bütün haftalar oynandı")
	}
	return s.playWeek(week)
}

func (s *leagueService) PlayAll() ([]models.Match, error) {
	var allMatches []models.Match
	for {
		week, err := s.getCurrentWeek()
		if err != nil {
			return nil, err
		}
		if week == 0 {
			break
		}
		played, err := s.playWeek(week)
		if err != nil {
			return nil, err
		}
		for _, m := range played {
			allMatches = append(allMatches, m)
		}
	}
	return allMatches, nil
}

func (s *leagueService) GetPredictions() ([]models.Prediction, error) {
	week, err := s.getCurrentWeek()
	if err != nil {
		return nil, err
	}
	
	playedWeeks := 6
	if week > 0 {
		playedWeeks = week - 1
	}
	
	if playedWeeks < 4 {
		return nil, errors.New("tahminler 4. haftadan sonra açılır")
	}
	
	teams, err := s.GetStandings()
	if err != nil {
		return nil, err
	}
	
	return calculatePredictions(teams), nil
}

func (s *leagueService) GetAllMatches() (map[int][]models.Match, error) {
	rows, err := s.db.Query(`
		SELECT m.id, m.week, m.home_id, m.away_id,
		       m.home_score, m.away_score, m.played,
		       ht.name, at.name
		FROM matches m
		JOIN teams ht ON ht.id = m.home_id
		JOIN teams at ON at.id = m.away_id
		ORDER BY m.week, m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]models.Match)
	for rows.Next() {
		var m models.Match
		var played int
		err := rows.Scan(
			&m.ID, &m.Week, &m.HomeID, &m.AwayID,
			&m.HomeScore, &m.AwayScore, &played,
			&m.HomeTeam, &m.AwayTeam,
		)
		if err != nil {
			return nil, err
		}
		if played == 1 {
			m.Played = true
		}
		result[m.Week] = append(result[m.Week], m)
	}
	return result, nil
}

func (s *leagueService) GetWeekSummary(week int) (*models.WeekSummary, error) {
	if week < 1 || week > 6 {
		return nil, errors.New("hafta 1 ile 6 arasında olmalı")
	}

	standings, err := s.GetStandings()
	if err != nil {
		return nil, err
	}

	allMatches, err := s.GetAllMatches()
	if err != nil {
		return nil, err
	}
	weekMatches := allMatches[week]

	current, err := s.getCurrentWeek()
	if err != nil {
		return nil, err
	}

	playedWeeks := 6
	if current > 0 {
		playedWeeks = current - 1
	}

	summary := &models.WeekSummary{
		Week:      week,
		Standings: standings,
		Matches:   weekMatches,
	}

	if playedWeeks >= 4 {
		summary.Predictions = calculatePredictions(standings)
	}

	return summary, nil
}

func (s *leagueService) EditMatchResult(matchID, homeScore, awayScore int) error {
	row := s.db.QueryRow(`
		SELECT id, week, home_id, away_id, home_score, away_score, played
		FROM matches WHERE id = ?`, matchID)
		
	var match models.Match
	var played int
	err := row.Scan(
		&match.ID, &match.Week, &match.HomeID, &match.AwayID,
		&match.HomeScore, &match.AwayScore, &played,
	)
	if err != nil {
		return err
	}
	
	if played == 0 {
		return errors.New("oynanmamış maç değiştirilemez")
	}

	teams, err := s.GetStandings()
	if err != nil {
		return err
	}
	
	var homeTeam, awayTeam models.Team
	for _, t := range teams {
		if t.ID == match.HomeID {
			homeTeam = t
		}
		if t.ID == match.AwayID {
			awayTeam = t
		}
	}

	homeTeam.Revert(*match.HomeScore, *match.AwayScore)
	awayTeam.Revert(*match.AwayScore, *match.HomeScore)
	homeTeam.Apply(homeScore, awayScore)
	awayTeam.Apply(awayScore, homeScore)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE teams SET played=?,won=?,drawn=?,lost=?,gf=?,ga=?,gd=?,points=? WHERE id=?`,
		homeTeam.Played, homeTeam.Won, homeTeam.Drawn, homeTeam.Lost, homeTeam.GF, homeTeam.GA, homeTeam.GD, homeTeam.Points, homeTeam.ID)
	if err != nil {
		return err
	}
	
	_, err = tx.Exec(`UPDATE teams SET played=?,won=?,drawn=?,lost=?,gf=?,ga=?,gd=?,points=? WHERE id=?`,
		awayTeam.Played, awayTeam.Won, awayTeam.Drawn, awayTeam.Lost, awayTeam.GF, awayTeam.GA, awayTeam.GD, awayTeam.Points, awayTeam.ID)
	if err != nil {
		return err
	}
	
	_, err = tx.Exec(`UPDATE matches SET home_score=?, away_score=?, played=1 WHERE id=?`, homeScore, awayScore, matchID)
	if err != nil {
		return err
	}
	
	return tx.Commit()
}

func (s *leagueService) playWeek(week int) ([]models.Match, error) {
	rows, err := s.db.Query(`
		SELECT m.id, m.week, m.home_id, m.away_id,
		       m.home_score, m.away_score, m.played,
		       ht.name, at.name
		FROM matches m
		JOIN teams ht ON ht.id = m.home_id
		JOIN teams at ON at.id = m.away_id
		WHERE m.week = ? AND m.played = 0`, week)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fixtures []models.Match
	for rows.Next() {
		var m models.Match
		var played int
		err := rows.Scan(
			&m.ID, &m.Week, &m.HomeID, &m.AwayID,
			&m.HomeScore, &m.AwayScore, &played,
			&m.HomeTeam, &m.AwayTeam,
		)
		if err != nil {
			return nil, err
		}
		fixtures = append(fixtures, m)
	}

	teams, err := s.GetStandings()
	if err != nil {
		return nil, err
	}
	
	teamMap := make(map[int]models.Team)
	for _, t := range teams {
		teamMap[t.ID] = t
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	randObj := rand.New(rand.NewSource(time.Now().UnixNano()))
	var played []models.Match

	for _, f := range fixtures {
		home := teamMap[f.HomeID]
		away := teamMap[f.AwayID]

		homeGoal := 0
		awayGoal := 0
		
		homeAttackProb := float64(home.Strength + 5) / 100.0
		awayAttackProb := float64(away.Strength) / 100.0
		
		for i := 0; i < 5; i++ {
			if randObj.Float64() < homeAttackProb {
				homeGoal++
			}
			if randObj.Float64() < awayAttackProb {
				awayGoal++
			}
		}

		home.Apply(homeGoal, awayGoal)
		away.Apply(awayGoal, homeGoal)
		teamMap[f.HomeID] = home
		teamMap[f.AwayID] = away

		_, err = tx.Exec(`UPDATE teams SET played=?,won=?,drawn=?,lost=?,gf=?,ga=?,gd=?,points=? WHERE id=?`,
			home.Played, home.Won, home.Drawn, home.Lost, home.GF, home.GA, home.GD, home.Points, home.ID)
		if err != nil {
			return nil, err
		}
			
		_, err = tx.Exec(`UPDATE teams SET played=?,won=?,drawn=?,lost=?,gf=?,ga=?,gd=?,points=? WHERE id=?`,
			away.Played, away.Won, away.Drawn, away.Lost, away.GF, away.GA, away.GD, away.Points, away.ID)
		if err != nil {
			return nil, err
		}
			
		_, err = tx.Exec(`UPDATE matches SET home_score=?, away_score=?, played=1 WHERE id=?`, homeGoal, awayGoal, f.ID)
		if err != nil {
			return nil, err
		}

		f.HomeScore = &homeGoal
		f.AwayScore = &awayGoal
		f.Played = true
		played = append(played, f)
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	
	return played, nil
}

func calculatePredictions(teams []models.Team) []models.Prediction {
	var entries []models.Prediction
	totalScore := 0.0
	
	for _, t := range teams {
		score := float64(t.Points)*10 + float64(t.GD)*2 + float64(t.Strength)
		if score < 0 {
			score = 0
		}
		totalScore += score
		
		p := models.Prediction{
			TeamName:   t.Name,
			Points:     t.Points,
			Percentage: score,
		}
		entries = append(entries, p)
	}

	for i := 0; i < len(entries); i++ {
		if totalScore > 0 {
			pct := (entries[i].Percentage / totalScore) * 100
			entries[i].Percentage = math.Round(pct*100) / 100
		} else {
			entries[i].Percentage = 0
		}
	}
	
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Percentage > entries[j].Percentage
	})
	
	return entries
}

func (s *leagueService) createFixtures() error {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM matches`).Scan(&count)
	if err != nil {
		return err
	}
	
	if count > 0 {
		return nil
	}

	rows, err := s.db.Query(`SELECT id FROM teams ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var teamIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		teamIDs = append(teamIDs, id)
	}

	n := len(teamIDs)
	for round := 0; round < 2; round++ {
		list := make([]int, n)
		copy(list, teamIDs)
		week := round*(n-1) + 1
		
		for r := 0; r < n-1; r++ {
			for i := 0; i < n/2; i++ {
				h := list[i]
				a := list[n-1-i]
				if round == 1 {
					temp := h
					h = a
					a = temp
				}
				s.db.Exec(`INSERT INTO matches (week, home_id, away_id) VALUES (?, ?, ?)`, week, h, a)
			}
			last := list[n-1]
			copy(list[2:], list[1:n-1])
			list[1] = last
			week++
		}
	}
	return nil
}

func (s *leagueService) getCurrentWeek() (int, error) {
	var week sql.NullInt64
	err := s.db.QueryRow(`SELECT MIN(week) FROM matches WHERE played = 0`).Scan(&week)
	if err != nil {
		return 0, err
	}
	if !week.Valid {
		return 0, nil
	}
	return int(week.Int64), nil
}
