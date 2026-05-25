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
