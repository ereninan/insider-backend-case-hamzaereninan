package models

type Team struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Strength int    `json:"strength"`
	Stats
}

type Stats struct {
	Played int `json:"played"`
	Won    int `json:"won"`
	Drawn  int `json:"drawn"`
	Lost   int `json:"lost"`
	GF     int `json:"gf"`
	GA     int `json:"ga"`
	GD     int `json:"gd"`
	Points int `json:"points"`
}

func (s *Stats) Apply(goalsFor, goalsAgainst int) {
	s.Played++
	s.GF += goalsFor
	s.GA += goalsAgainst
	s.GD = s.GF - s.GA

	switch {
	case goalsFor > goalsAgainst:
		s.Won++
		s.Points += 3
	case goalsFor == goalsAgainst:
		s.Drawn++
		s.Points++
	default:
		s.Lost++
	}
}

func (s *Stats) Revert(goalsFor, goalsAgainst int) {
	s.Played--
	s.GF -= goalsFor
	s.GA -= goalsAgainst
	s.GD = s.GF - s.GA

	switch {
	case goalsFor > goalsAgainst:
		s.Won--
		s.Points -= 3
	case goalsFor == goalsAgainst:
		s.Drawn--
		s.Points--
	default:
		s.Lost--
	}
}

type Match struct {
	ID        int    `json:"id"`
	Week      int    `json:"week"`
	HomeID    int    `json:"home_id"`
	AwayID    int    `json:"away_id"`
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeScore *int   `json:"home_score"`
	AwayScore *int   `json:"away_score"`
	Played    bool   `json:"played"`
}

type Prediction struct {
	TeamName   string  `json:"team_name"`
	Points     int     `json:"points"`
	Percentage float64 `json:"percentage"`
}

type WeekSummary struct {
	Week        int          `json:"week"`
	Standings   []Team       `json:"standings"`
	Matches     []Match      `json:"matches"`
	Predictions []Prediction `json:"predictions,omitempty"`
}
