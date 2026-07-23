CREATE TABLE IF NOT EXISTS abbreviations (
    id SERIAL PRIMARY KEY,
    abbreviation TEXT NOT NULL,
    full_word TEXT NOT NULL,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_abbreviations_abbr ON abbreviations(UPPER(abbreviation));
