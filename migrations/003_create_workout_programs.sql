-- Migration: Création des tables workout programs
-- Date: 2025-10-07

-- Table principale des programmes d'entraînement
CREATE TABLE IF NOT EXISTS workout_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- FREE_MODE, TARGET_REPS, MAX_TIME, SETS_REPS, PYRAMID, EMOM, AMRAP
    variant VARCHAR(50) NOT NULL, -- STANDARD, INCLINE, DECLINE, DIAMOND, WIDE, PIKE, ARCHER
    difficulty VARCHAR(50) NOT NULL, -- BEGINNER, INTERMEDIATE, ADVANCED
    rest_between_sets INTEGER,

    -- Champs spécifiques selon le type
    target_reps INTEGER, -- Pour TARGET_REPS
    time_limit INTEGER, -- Pour TARGET_REPS (optionnel)
    duration INTEGER, -- Pour MAX_TIME, AMRAP (en secondes)
    allow_rest BOOLEAN, -- Pour MAX_TIME
    sets INTEGER, -- Pour SETS_REPS
    reps_per_set INTEGER, -- Pour SETS_REPS
    reps_sequence JSONB, -- Pour PYRAMID, tableau d'entiers ex: [5, 10, 15, 10, 5]
    reps_per_minute INTEGER, -- Pour EMOM
    total_minutes INTEGER, -- Pour EMOM

    is_custom BOOLEAN DEFAULT false,
    is_featured BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0,

    created_by UUID,
    updated_by UUID,
    deleted_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Table des sessions d'entraînement
CREATE TABLE IF NOT EXISTS workout_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES workout_programs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    total_reps INTEGER DEFAULT 0,
    total_duration INTEGER DEFAULT 0, -- en secondes
    completed BOOLEAN DEFAULT false,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Table des résultats de séries
CREATE TABLE IF NOT EXISTS set_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES workout_sessions(id) ON DELETE CASCADE,
    set_number INTEGER NOT NULL,
    target_reps INTEGER,
    completed_reps INTEGER NOT NULL,
    duration INTEGER NOT NULL, -- en secondes
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index pour améliorer les performances
CREATE INDEX idx_programs_type ON workout_programs(type);
CREATE INDEX idx_programs_variant ON workout_programs(variant);
CREATE INDEX idx_programs_difficulty ON workout_programs(difficulty);
CREATE INDEX idx_programs_is_custom ON workout_programs(is_custom);
CREATE INDEX idx_programs_is_featured ON workout_programs(is_featured);
CREATE INDEX idx_programs_usage_count ON workout_programs(usage_count DESC);
CREATE INDEX idx_programs_deleted_at ON workout_programs(deleted_at);
CREATE INDEX idx_programs_created_at ON workout_programs(created_at DESC);

CREATE INDEX idx_sessions_program_id ON workout_sessions(program_id);
CREATE INDEX idx_sessions_user_id ON workout_sessions(user_id);
CREATE INDEX idx_sessions_start_time ON workout_sessions(start_time DESC);
CREATE INDEX idx_sessions_completed ON workout_sessions(completed);

CREATE INDEX idx_set_results_session_id ON set_results(session_id);
CREATE INDEX idx_set_results_set_number ON set_results(set_number);

-- Contraintes de clés étrangères pour created_by, updated_by, deleted_by
ALTER TABLE workout_programs
ADD CONSTRAINT fk_programs_created_by FOREIGN KEY (created_by) REFERENCES users(id),
ADD CONSTRAINT fk_programs_updated_by FOREIGN KEY (updated_by) REFERENCES users(id),
ADD CONSTRAINT fk_programs_deleted_by FOREIGN KEY (deleted_by) REFERENCES users(id);

-- Insertion de quelques programmes par défaut pour commencer
INSERT INTO workout_programs (name, description, type, variant, difficulty, target_reps, is_featured) VALUES
('Débutant - 20 Pompes', 'Parfait pour commencer votre parcours fitness', 'TARGET_REPS', 'STANDARD', 'BEGINNER', 20, true),
('Intermédiaire - 50 Pompes', 'Challenge intermédiaire pour progresser', 'TARGET_REPS', 'STANDARD', 'INTERMEDIATE', 50, true),
('Avancé - 100 Pompes', 'Pour les athlètes confirmés', 'TARGET_REPS', 'STANDARD', 'ADVANCED', 100, true),
('3x10 Pompes Standard', 'Programme classique en séries', 'SETS_REPS', 'STANDARD', 'BEGINNER', NULL, true);

UPDATE workout_programs SET sets=3, reps_per_set=10, rest_between_sets=60 WHERE name='3x10 Pompes Standard';
