CREATE TABLE habit_entries (
    id SERIAL PRIMARY KEY,
    habit_id INT REFERENCES habits(id) ON DELETE CASCADE,
    entry_date DATE NOT NULL, -- Date of check-in
    status VARCHAR(20) NOT NULL, -- 'completed', 'skipped', 'missed'
    notes TEXT, -- Optional user notes
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (habit_id, entry_date) -- Prevent duplicate entries
);