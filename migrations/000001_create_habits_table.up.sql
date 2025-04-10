CREATE TABLE habits (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    frequency VARCHAR(50) NOT NULL, -- 'daily', 'weekly', 'custom'
    goal VARCHAR(100), -- e.g., "3 times/week"
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);