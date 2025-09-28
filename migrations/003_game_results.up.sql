CREATE TABLE IF NOT EXISTS game_results (
    id SERIAL PRIMARY KEY,
    game_id INT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    player_id BIGINT NOT NULL REFERENCES players(tg_id) ON DELETE CASCADE,
    score_change INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
