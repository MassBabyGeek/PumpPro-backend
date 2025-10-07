-- Migration: Création des tables pour le leaderboard
-- Date: 2025-10-07

-- Table de cache pour le leaderboard (optionnelle, pour optimiser les performances)
CREATE TABLE IF NOT EXISTS leaderboard_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period VARCHAR(20) NOT NULL, -- daily, weekly, monthly, all-time
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score INTEGER NOT NULL DEFAULT 0,
    rank INTEGER NOT NULL,
    change INTEGER DEFAULT 0, -- Changement par rapport à la période précédente
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(period, user_id)
);

-- Table des badges utilisateurs
CREATE TABLE IF NOT EXISTS user_badges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_code VARCHAR(50) NOT NULL, -- Code du badge (ex: 'top_1', 'streak_7', etc.)
    badge_emoji VARCHAR(10), -- Emoji du badge (ex: '👑', '🔥', etc.)
    earned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, badge_code)
);

-- Table des relations d'amitié (pour le leaderboard entre amis)
CREATE TABLE IF NOT EXISTS user_friendships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, accepted, blocked
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, friend_id),
    CHECK (user_id != friend_id)
);

-- Index pour améliorer les performances
CREATE INDEX idx_leaderboard_cache_period ON leaderboard_cache(period);
CREATE INDEX idx_leaderboard_cache_rank ON leaderboard_cache(period, rank);
CREATE INDEX idx_leaderboard_cache_user_id ON leaderboard_cache(user_id);
CREATE INDEX idx_leaderboard_cache_updated_at ON leaderboard_cache(updated_at DESC);

CREATE INDEX idx_user_badges_user_id ON user_badges(user_id);
CREATE INDEX idx_user_badges_badge_code ON user_badges(badge_code);
CREATE INDEX idx_user_badges_earned_at ON user_badges(earned_at DESC);

CREATE INDEX idx_friendships_user_id ON user_friendships(user_id);
CREATE INDEX idx_friendships_friend_id ON user_friendships(friend_id);
CREATE INDEX idx_friendships_status ON user_friendships(status);

-- Fonction pour rafraîchir le cache du leaderboard (à appeler périodiquement)
CREATE OR REPLACE FUNCTION refresh_leaderboard_cache(p_period VARCHAR)
RETURNS void AS $$
DECLARE
    v_start_date TIMESTAMP;
BEGIN
    -- Déterminer la date de début selon la période
    CASE p_period
        WHEN 'daily' THEN
            v_start_date := CURRENT_DATE;
        WHEN 'weekly' THEN
            v_start_date := CURRENT_DATE - INTERVAL '7 days';
        WHEN 'monthly' THEN
            v_start_date := CURRENT_DATE - INTERVAL '30 days';
        WHEN 'all-time' THEN
            v_start_date := '1970-01-01'::TIMESTAMP;
        ELSE
            v_start_date := '1970-01-01'::TIMESTAMP;
    END CASE;

    -- Supprimer l'ancien cache pour cette période
    DELETE FROM leaderboard_cache WHERE period = p_period;

    -- Calculer et insérer le nouveau classement
    INSERT INTO leaderboard_cache (period, user_id, score, rank, updated_at)
    SELECT
        p_period,
        user_id,
        score,
        rank,
        NOW()
    FROM (
        SELECT
            ws.user_id,
            SUM(ws.total_reps) as score,
            ROW_NUMBER() OVER (ORDER BY SUM(ws.total_reps) DESC) as rank
        FROM workout_sessions ws
        WHERE ws.start_time >= v_start_date
        GROUP BY ws.user_id
    ) ranked_users;
END;
$$ LANGUAGE plpgsql;

-- Insertion de quelques badges par défaut
INSERT INTO user_badges (user_id, badge_code, badge_emoji) VALUES
((SELECT id FROM users LIMIT 1), 'first_workout', '🌟')
ON CONFLICT (user_id, badge_code) DO NOTHING;

-- Note: Pour automatiser le rafraîchissement du cache, vous pouvez utiliser pg_cron ou un cron job externe
-- Exemple: SELECT refresh_leaderboard_cache('daily');
