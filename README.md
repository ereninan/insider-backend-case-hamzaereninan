# Football League Simulator

A production-grade REST API that simulates a Premier League-style football season for 4 clubs, built with Go and SQLite.

---

## Table of Contents

- [Overview](#overview)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Architectural Choices](#architectural-choices)
- [Championship Prediction Algorithm](#championship-prediction-algorithm)
- [API Reference](#api-reference)

---

## Overview

The simulator manages a 6-week double round-robin league between **Manchester City**, **Liverpool**, **Arsenal**, and **Chelsea**. Each club has a base **Strength** rating that drives probabilistic match simulation. The system persists all state in a local SQLite database and exposes a clean JSON REST API.

---

## Project Structure

```
football-league-simulator/
├── cmd/
│   └── server/
│       └── main.go              # Wires DB → Service → Handlers → HTTP server
├── internal/
│   ├── db/
│   │   └── db.go                # SQLite connection, schema bootstrap
│   ├── models/
│   │   └── models.go            # Team (with embedded Stats), Match, Prediction
│   ├── service/
│   │   └── service.go           # LeagueManager interface + full implementation
│   └── handlers/
│       └── handlers.go          # HTTP handlers, chi router, JSON helpers
├── schema/
│   └── schema.sql               # DDL + seed INSERT for 4 teams
├── go.mod
└── go.sum
```

---

## Getting Started

**Prerequisites:** Go 1.21+ and a C compiler (required for SQLite's CGo driver).

```bash
# macOS
xcode-select --install

# Clone and run
git clone <repo-url>
cd football-league-simulator
go run cmd/server/main.go
```

The server starts on **http://localhost:8080**. A `league.db` file is created automatically in the working directory.

---

## Architectural Choices

### Interface-Based Design

Every layer is hidden behind an interface:

```go
type LeagueManager interface {
    PlayNextWeek() ([]models.Match, error)
    PlayAll()      ([]models.Match, error)
    GetStandings() ([]models.Team, error)
    GetPredictions() ([]models.Prediction, error)
    EditMatchResult(matchID, homeScore, awayScore int) error
}
```

The `handlers` package depends only on `LeagueManager` — it never imports the concrete `leagueManager` struct. This makes the entire service layer trivially mockable in tests.

### Struct Composition

`Team` embeds `Stats`, promoting all statistical fields and the `Apply`/`Revert` methods into `Team`'s method set:

```go
type Team struct {
    ID, Name, Strength ...
    Stats               // embedded — promotes Apply() and Revert()
}

type Stats struct {
    Played, Won, Drawn, Lost int
    GF, GA, GD, Points       int
}

func (s *Stats) Apply(goalsFor, goalsAgainst int)  { ... }
func (s *Stats) Revert(goalsFor, goalsAgainst int) { ... }
```

This ensures point arithmetic lives in exactly one place and can never diverge between play and edit paths.

### Match Simulation

Each team gets `attackRounds = 5` chances to score. Each chance succeeds with probability `strength / 100` (home team gets +5 strength bonus). This produces realistic 0–5 goal ranges weighted by relative team quality.

### Transactional Consistency

`PlayNextWeek` and `EditMatchResult` both operate inside a single `*sql.Tx`. If any update fails, the entire week's changes roll back atomically.

### SQLite Concurrency

`SetMaxOpenConns(1)` prevents write-lock contention. To avoid nested-query deadlocks (rows cursor open while querying for individual teams), all data is loaded into memory in a single query before the transaction begins.

---

## Championship Prediction Algorithm

Available from **Week 4 onwards**.

Each team receives a weighted score:

```
score = (points × 10) + (goal_difference × 2) + (strength × 1)
```

| Factor         | Weight | Rationale                                         |
|----------------|--------|---------------------------------------------------|
| Points         | ×10    | Directly determines league position               |
| Goal Difference| ×2     | Official Premier League tiebreaker                |
| Strength       | ×1     | Potential to recover points in remaining fixtures |

Championship probability:

```
probability(team) = score(team) / Σ score(all teams) × 100%
```

All percentages sum to exactly 100%. Teams with a negative weighted score are floored to 0 before normalisation.

---

## API Reference

### GET /api/standings

Returns the current league table, sorted by Points → GD → GF.

```bash
curl http://localhost:8080/api/standings
```

```json
[
  { "id": 1, "name": "Manchester City", "strength": 90,
    "played": 4, "won": 3, "drawn": 1, "lost": 0,
    "gf": 14, "ga": 7, "gd": 7, "points": 10 },
  ...
]
```

---

### GET /api/matches

Returns all fixtures grouped by week. Unplayed matches show `home_score: null`.

```bash
curl http://localhost:8080/api/matches
```

```json
{
  "1": [
    { "id": 1, "week": 1, "home_team": "Manchester City", "away_team": "Chelsea",
      "home_score": 3, "away_score": 1, "played": true }
  ],
  "2": [
    { "id": 3, "week": 2, "home_team": "Arsenal", "away_team": "Liverpool",
      "home_score": null, "away_score": null, "played": false }
  ]
}
```

---

### GET /api/week/{n}

The core case view: league table + that week's matches. Includes `predictions` if at least 4 weeks have been played.

```bash
curl http://localhost:8080/api/week/4
```

```json
{
  "week": 4,
  "standings": [ ... ],
  "matches": [
    { "id": 7, "week": 4, "home_team": "Chelsea", "away_team": "Manchester City",
      "home_score": 2, "away_score": 3, "played": true }
  ],
  "predictions": [
    { "team_name": "Manchester City", "points": 10, "percentage": 38.42 },
    { "team_name": "Liverpool",       "points": 7,  "percentage": 28.17 }
  ]
}
```

`predictions` is omitted (not present in JSON) before week 4 is completed.

---

### POST /api/simulate-week

Simulates all unplayed matches in the next scheduled week.

```bash
curl -X POST http://localhost:8080/api/simulate-week
```

```json
{
  "played": [
    { "id": 1, "week": 1,
      "home_team": "Manchester City", "away_team": "Chelsea",
      "home_score": 4, "away_score": 2, "played": true }
  ]
}
```

Returns `400` with `{ "error": "all weeks have been played" }` when the season is over.

---

### POST /api/simulate-all

Simulates all remaining unplayed weeks in one request.

```bash
curl -X POST http://localhost:8080/api/simulate-all
```

Response: same shape as `/api/simulate-week`, containing all played matches across all remaining weeks.

---

### GET /api/predictions

Returns championship probability for each team. Only available after Week 4 has been played.

```bash
curl http://localhost:8080/api/predictions
```

```json
[
  { "team_name": "Manchester City", "points": 10, "percentage": 38.42 },
  { "team_name": "Liverpool",       "points": 7,  "percentage": 28.17 },
  { "team_name": "Arsenal",         "points": 5,  "percentage": 19.55 },
  { "team_name": "Chelsea",         "points": 3,  "percentage": 13.86 }
]
```

Before Week 4:

```json
{ "error": "predictions are available from week 4 onwards (currently week 2)" }
```

---

### PUT /api/match/{id}

Edits the result of a played match and recalculates affected team stats atomically.

```bash
curl -X PUT http://localhost:8080/api/match/1 \
  -H "Content-Type: application/json" \
  -d '{"home_score": 3, "away_score": 0}'
```

```json
{ "status": "match result updated" }
```

The system reverts the old score's contribution (`Stats.Revert`), applies the new score (`Stats.Apply`), and persists both team stats and the new match score in a single transaction.

Returns `400` if the match has not been played yet.
