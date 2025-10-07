-- Migration: Création des tables challenges
-- Date: 2025-10-07

-- Table principale des challenges
CREATE TABLE IF NOT EXISTS challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL, -- DAILY, WEEKLY, MONTHLY, etc.
    type VARCHAR(50) NOT NULL,
    variant VARCHAR(100),
    difficulty VARCHAR(50) NOT NULL, -- BEGINNER, INTERMEDIATE, ADVANCED
    target_reps INTEGER,
    duration INTEGER,
    sets INTEGER,
    reps_per_set INTEGER,
    image_url TEXT,
    icon_name VARCHAR(100) NOT NULL,
    icon_color VARCHAR(50) NOT NULL,
    participants INTEGER DEFAULT 0,
    completions INTEGER DEFAULT 0,
    likes INTEGER DEFAULT 0,
    points INTEGER DEFAULT 0,
    badge VARCHAR(100),
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, UPCOMING, COMPLETED, EXPIRED
    tags TEXT[] DEFAULT '{}',
    is_official BOOLEAN DEFAULT false,
    created_by UUID,
    updated_by UUID,
    deleted_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Table de progression des utilisateurs sur les challenges
CREATE TABLE IF NOT EXISTS user_challenge_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    progress INTEGER DEFAULT 0, -- pourcentage de 0 à 100
    current_reps INTEGER DEFAULT 0,
    target_reps INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(challenge_id, user_id)
);

-- Table des likes de challenges
CREATE TABLE IF NOT EXISTS challenge_likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(challenge_id, user_id)
);

-- Index pour améliorer les performances
CREATE INDEX idx_challenges_category ON challenges(category);
CREATE INDEX idx_challenges_difficulty ON challenges(difficulty);
CREATE INDEX idx_challenges_type ON challenges(type);
CREATE INDEX idx_challenges_status ON challenges(status);
CREATE INDEX idx_challenges_deleted_at ON challenges(deleted_at);
CREATE INDEX idx_challenges_start_date ON challenges(start_date);
CREATE INDEX idx_challenges_end_date ON challenges(end_date);
CREATE INDEX idx_challenges_created_at ON challenges(created_at);
CREATE INDEX idx_challenges_tags ON challenges USING GIN(tags);

CREATE INDEX idx_user_progress_user_id ON user_challenge_progress(user_id);
CREATE INDEX idx_user_progress_challenge_id ON user_challenge_progress(challenge_id);
CREATE INDEX idx_user_progress_progress ON user_challenge_progress(progress);
CREATE INDEX idx_user_progress_completed_at ON user_challenge_progress(completed_at);

CREATE INDEX idx_challenge_likes_challenge_id ON challenge_likes(challenge_id);
CREATE INDEX idx_challenge_likes_user_id ON challenge_likes(user_id);

-- Contraintes de clés étrangères pour created_by, updated_by, deleted_by
ALTER TABLE challenges
ADD CONSTRAINT fk_challenges_created_by FOREIGN KEY (created_by) REFERENCES users(id),
ADD CONSTRAINT fk_challenges_updated_by FOREIGN KEY (updated_by) REFERENCES users(id),
ADD CONSTRAINT fk_challenges_deleted_by FOREIGN KEY (deleted_by) REFERENCES users(id);
