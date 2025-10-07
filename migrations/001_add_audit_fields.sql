-- Migration: Ajout des champs d'audit (createdBy, updatedBy, deletedAt, deletedBy)
-- Date: 2025-10-07

-- Ajout des colonnes d'audit à la table users
ALTER TABLE users
ADD COLUMN created_by UUID,
ADD COLUMN updated_by UUID,
ADD COLUMN deleted_at TIMESTAMP,
ADD COLUMN deleted_by UUID;

-- Ajout des index pour améliorer les performances des requêtes
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_created_by ON users(created_by);
CREATE INDEX idx_users_updated_by ON users(updated_by);

-- Ajout des colonnes d'audit à la table sessions
ALTER TABLE sessions
ADD COLUMN created_by UUID,
ADD COLUMN updated_by UUID,
ADD COLUMN deleted_at TIMESTAMP,
ADD COLUMN deleted_by UUID;

-- Ajout des index pour la table sessions
CREATE INDEX idx_sessions_deleted_at ON sessions(deleted_at);
CREATE INDEX idx_sessions_created_by ON sessions(created_by);

-- Ajout des clés étrangères pour assurer l'intégrité référentielle
ALTER TABLE users
ADD CONSTRAINT fk_users_created_by FOREIGN KEY (created_by) REFERENCES users(id),
ADD CONSTRAINT fk_users_updated_by FOREIGN KEY (updated_by) REFERENCES users(id),
ADD CONSTRAINT fk_users_deleted_by FOREIGN KEY (deleted_by) REFERENCES users(id);

ALTER TABLE sessions
ADD CONSTRAINT fk_sessions_created_by FOREIGN KEY (created_by) REFERENCES users(id),
ADD CONSTRAINT fk_sessions_updated_by FOREIGN KEY (updated_by) REFERENCES users(id),
ADD CONSTRAINT fk_sessions_deleted_by FOREIGN KEY (deleted_by) REFERENCES users(id);
