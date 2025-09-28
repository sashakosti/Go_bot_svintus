CREATE TABLE IF NOT EXISTS players (
    tg_id BIGINT PRIMARY KEY,
    username TEXT,
    display_name TEXT,
    score INT DEFAULT 0
);

