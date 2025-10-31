# Changelog - PumpPro Backend

## [Dernière Version] - 2025-10-31

### 🎉 Nouvelles Fonctionnalités Majeures

#### 1. **Intégration Cloudinary pour le Stockage d'Images**
- ✅ Upload d'avatars vers Cloudinary (stockage persistant)
- ✅ Upload d'images de challenges vers Cloudinary
- ✅ Upload de screenshots de bug reports vers Cloudinary
- ✅ Fallback automatique vers stockage local si Cloudinary non configuré
- ✅ Optimisation automatique des images (compression, resize, format)
- ✅ CDN global pour chargement rapide

**Fichiers ajoutés:**
- `internal/services/cloudinary.go` - Service Cloudinary
- `CLOUDINARY_SETUP.md` - Guide de configuration
- `STORAGE.md` - Explication du problème de stockage éphémère

**Fichiers modifiés:**
- `internal/config/config.go` - Ajout des credentials Cloudinary
- `internal/handler/user.go` - Upload vers Cloudinary dans UploadAvatar
- `environment/.env` - Ajout des variables Cloudinary
- `environment/.env.example` - Documentation des variables

#### 2. **Dashboard Admin Complet**
Routes ajoutées pour l'administration:

**Statistiques:**
- `GET /admin/dashboard` - Statistiques globales de l'app
- `GET /admin/activity` - Activité récente des utilisateurs
- `GET /admin/health` - Santé du système
- `GET /admin/top-content` - Contenus les plus populaires
- `GET /admin/analytics?period=30d` - Analytics avec graphiques

**Gestion Utilisateurs:**
- `GET /admin/users` - Liste des utilisateurs (recherche, tri, pagination)
- `PUT/PATCH /admin/users/{userId}` - Modifier un utilisateur (tous champs)
- `DELETE /admin/users/{userId}` - Supprimer un utilisateur
- `POST /admin/users/{userId}/promote` - Promouvoir en admin
- `POST /admin/users/{userId}/demote` - Retirer privilèges admin

**Gestion Contenu:**
- `GET /admin/photos` - Liste de toutes les photos de l'app

**Fichiers ajoutés:**
- `internal/models/admin.go` - Modèles pour le dashboard admin
- `internal/handler/admin.go` - Handlers admin (enrichis)

**Fichiers modifiés:**
- `internal/api/routes.go` - Routes admin ajoutées

### 🔧 Corrections de Bugs

#### 1. **Fix CORS pour les Images**
- ✅ Les avatars fonctionnent maintenant depuis le navigateur et l'admin web
- ✅ Ajout de la méthode OPTIONS pour les requêtes preflight
- ✅ Headers CORS explicites dans GetAvatar

**Fichiers modifiés:**
- `internal/middleware/cors.go` - Ajout de PUT, PATCH, DELETE
- `internal/handler/user.go` - Headers CORS dans GetAvatar
- `internal/api/routes.go` - Méthode OPTIONS sur /avatars/{filename}

#### 2. **Fix Admin User Management**
- ✅ Routes admin séparées pour éviter les conflits
- ✅ Modification utilisateur accepte le champ `password`
- ✅ Suppression utilisateur cible le bon utilisateur (pas l'admin connecté)
- ✅ Empêche un admin de se supprimer lui-même

**Routes corrigées:**
- `PUT/PATCH /admin/users/{userId}` - Nouveau endpoint dédié admin
- `DELETE /admin/users/{userId}` - Nouveau endpoint dédié admin

#### 3. **Fix Route /admin/photos**
- ✅ Retourne maintenant les photos avec `LENGTH(avatar) > 0` au lieu de `avatar != ''`
- ✅ Formatage correct des dates (RFC3339)
- ✅ Affiche les avatars existants dans la DB

**Fichiers modifiés:**
- `internal/handler/admin.go` - Requêtes SQL corrigées

### 📦 Dépendances Ajoutées

```bash
go get github.com/cloudinary/cloudinary-go/v2
go get github.com/creasty/defaults
go get github.com/gorilla/schema
```

### ⚙️ Configuration Requise

**Nouvelles Variables d'Environnement:**
```env
CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=your_api_key
CLOUDINARY_API_SECRET=your_api_secret
```

### 🚀 Déploiement

1. **Créer un compte Cloudinary** (gratuit) sur [cloudinary.com](https://cloudinary.com)
2. **Récupérer les credentials** depuis le Dashboard
3. **Configurer les variables** dans Render.com:
   - Environment → Environment Variables
   - Ajouter les 3 variables Cloudinary
4. **Redéployer** - Les nouvelles images iront automatiquement vers Cloudinary

### 📊 Métriques Dashboard Admin

Le dashboard admin fournit:
- Total utilisateurs, utilisateurs actifs (24h)
- Nouveaux utilisateurs (jour, semaine, mois)
- Total challenges, challenges actifs
- Total programmes, workouts, pompes
- Workouts (jour, semaine, mois)
- Bug reports (total, en attente)
- Moyennes (pompes/utilisateur, workouts/utilisateur)
- Total photos stockées
- Utilisation du stockage

### 🔐 Sécurité

- ✅ Toutes les routes admin vérifient `middleware.IsAdmin(r)`
- ✅ Les credentials Cloudinary sont dans les variables d'environnement
- ✅ Pas de credentials hardcodés dans le code
- ✅ Protection contre la suppression par un admin de lui-même

### 📝 Documentation

**Nouveaux Fichiers:**
- `CLOUDINARY_SETUP.md` - Guide de configuration Cloudinary
- `STORAGE.md` - Explication du problème de stockage sur Render
- `environment/.env.example` - Template pour .env
- `CHANGELOG.md` - Ce fichier

### ⚠️ Breaking Changes

Aucun breaking change - Backward compatible:
- Si Cloudinary n'est pas configuré, fallback vers stockage local
- Les anciennes URLs d'avatars continuent de fonctionner
- Les routes existantes restent inchangées

### 🎯 Prochaines Étapes Recommandées

1. **Configurer Cloudinary** pour éviter la perte d'images
2. **Migrer les avatars existants** vers Cloudinary (optionnel)
3. **Tester le dashboard admin** avec votre compte admin
4. **Monitorer les métriques** via `/admin/dashboard`

### 📞 Support

- Issues GitHub: [github.com/MassBabyGeek/PumpPro-backend/issues](https://github.com/MassBabyGeek/PumpPro-backend/issues)
- Documentation Cloudinary: [CLOUDINARY_SETUP.md](CLOUDINARY_SETUP.md)
- Problème de stockage: [STORAGE.md](STORAGE.md)

---

**Version:** v2.0.0
**Date:** 31 Octobre 2025
**Contributeur:** Claude AI + Lucas Usereau
