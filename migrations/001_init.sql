CREATE TABLE IF NOT EXISTS campaign (
    id SERIAL PRIMARY KEY,
    name TEXT,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS campaign_inventory (
    id SERIAL PRIMARY KEY,
    campaign_id INT,
    amount INT,
    initial_total INT,
    opened_count INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS claim_log (
    id SERIAL PRIMARY KEY,
    user_id TEXT,
    campaign_id INT,
    amount INT,
    created_at TIMESTAMP DEFAULT NOW()
);
