CREATE TABLE IF NOT EXISTS teams (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    name     TEXT    NOT NULL UNIQUE,
    strength INTEGER NOT NULL DEFAULT 50,
    played   INTEGER NOT NULL DEFAULT 0,
    won      INTEGER NOT NULL DEFAULT 0,
    drawn    INTEGER NOT NULL DEFAULT 0,
    lost     INTEGER NOT NULL DEFAULT 0,
    gf       INTEGER NOT NULL DEFAULT 0,
    ga       INTEGER NOT NULL DEFAULT 0,
    gd       INTEGER NOT NULL DEFAULT 0,
    points   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS matches (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    week       INTEGER NOT NULL,
    home_id    INTEGER NOT NULL REFERENCES teams(id),
    away_id    INTEGER NOT NULL REFERENCES teams(id),
    home_score INTEGER,
    away_score INTEGER,
    played     INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO teams (name, strength) VALUES
    ('Manchester City', 90),
    ('Liverpool',       85),
    ('Arsenal',         82),
    ('Chelsea',         78);

SELECT id, name, strength, played, won, drawn, lost, gf, ga, gd, points
FROM teams
ORDER BY points DESC, gd DESC, gf DESC, name ASC;

SELECT id FROM teams ORDER BY id;

SELECT id, name, strength, played, won, drawn, lost, gf, ga, gd, points
FROM teams;

SELECT m.id, m.week, m.home_id, m.away_id,
       m.home_score, m.away_score, m.played,
       ht.name AS home_team,
       at.name AS away_team
FROM matches m
JOIN teams ht ON ht.id = m.home_id
JOIN teams at ON at.id = m.away_id
WHERE m.week = ? AND m.played = 0
ORDER BY m.id;

SELECT m.id, m.week, m.home_id, m.away_id,
       m.home_score, m.away_score, m.played,
       ht.name AS home_team,
       at.name AS away_team
FROM matches m
JOIN teams ht ON ht.id = m.home_id
JOIN teams at ON at.id = m.away_id
WHERE m.week = ?
ORDER BY m.id;

SELECT m.id, m.week, m.home_id, m.away_id,
       m.home_score, m.away_score, m.played,
       ht.name AS home_team,
       at.name AS away_team
FROM matches m
JOIN teams ht ON ht.id = m.home_id
JOIN teams at ON at.id = m.away_id
ORDER BY m.week, m.id;

SELECT MIN(week) FROM matches WHERE played = 0;

SELECT id, week, home_id, away_id, home_score, away_score, played
FROM matches
WHERE id = ?;

SELECT COUNT(*) FROM matches;

INSERT INTO matches (week, home_id, away_id) VALUES (?, ?, ?);

UPDATE teams
SET played = ?,
    won    = ?,
    drawn  = ?,
    lost   = ?,
    gf     = ?,
    ga     = ?,
    gd     = ?,
    points = ?
WHERE id = ?;

UPDATE matches
SET home_score = ?,
    away_score = ?,
    played     = 1
WHERE id = ?;
