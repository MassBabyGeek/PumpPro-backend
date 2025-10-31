# Configuration Cloudinary pour PumpPro

## 📸 Pourquoi Cloudinary?

Cloudinary résout le problème du **stockage éphémère** sur Render.com:
- ✅ **Stockage persistant** - Les images ne sont jamais perdues
- ✅ **CDN global** - Chargement rapide partout dans le monde
- ✅ **Gratuit jusqu'à 25GB** - Parfait pour démarrer
- ✅ **Optimisation automatique** - Compression et transformation d'images
- ✅ **HTTPS inclus** - Sécurité par défaut

## 🚀 Configuration en 5 Minutes

### 1. Créer un Compte Cloudinary

1. Allez sur [https://cloudinary.com/users/register/free](https://cloudinary.com/users/register/free)
2. Créez un compte gratuit
3. Vérifiez votre email

### 2. Récupérer vos Credentials

1. Connectez-vous à [https://cloudinary.com/console](https://cloudinary.com/console)
2. Sur le Dashboard, vous verrez:
   ```
   Cloud Name: votre_cloud_name
   API Key: 123456789012345
   API Secret: abcd1234efgh5678ijkl
   ```

### 3. Configurer le .env

Ouvrez `environment/.env` et ajoutez:

```env
# Cloudinary Configuration
CLOUDINARY_CLOUD_NAME=votre_cloud_name
CLOUDINARY_API_KEY=123456789012345
CLOUDINARY_API_SECRET=abcd1234efgh5678ijkl
```

**⚠️ Important:** Ne commitez JAMAIS ces valeurs dans Git!

### 4. Configurer sur Render.com

Dans votre service Render:

1. Allez dans **Environment** → **Environment Variables**
2. Ajoutez les 3 variables:
   - `CLOUDINARY_CLOUD_NAME`
   - `CLOUDINARY_API_KEY`
   - `CLOUDINARY_API_SECRET`

### 5. Redéployer

```bash
git push origin main
```

Render redéploiera automatiquement avec Cloudinary activé!

## 🧪 Tester l'Intégration

### Test Local

1. Assurez-vous que `.env` contient vos credentials Cloudinary
2. Démarrez le serveur:
   ```bash
   go run cmd/server/main.go
   ```

3. Uploadez un avatar via l'API ou l'app mobile
4. Vérifiez dans votre [Dashboard Cloudinary](https://cloudinary.com/console/media_library) → dossier `pumppro/avatars`

### Vérification

Si Cloudinary est **configuré**, l'avatar sera uploadé vers:
```
https://res.cloudinary.com/votre_cloud_name/image/upload/pumppro/avatars/user_id.jpg
```

Si Cloudinary n'est **pas configuré**, fallback vers stockage local:
```
https://pumppro-backend.onrender.com/avatars/user_id.jpg
```
⚠️ **Attention:** Le stockage local sera perdu au prochain redéploiement!

## 📊 Comment Ça Fonctionne

### Upload d'Avatar

1. L'utilisateur upload une image via `/users/{id}/avatar`
2. Le backend vérifie si Cloudinary est configuré:
   - **OUI** → Upload vers Cloudinary
   - **NON** → Stockage local (éphémère)
3. L'URL est enregistrée dans la base de données
4. L'application affiche l'image via l'URL Cloudinary (directe, pas de proxy)

### Avantages Cloudinary

```go
// Transformation automatique appliquée
Transformation: "c_fill,g_face,h_500,w_500"
```

Cela signifie:
- **c_fill**: Remplit le cadre en coupant si nécessaire
- **g_face**: Centre sur le visage détecté automatiquement
- **h_500,w_500**: Redimensionne à 500x500 pixels
- **Format JPG**: Conversion automatique pour optimiser la taille

## 📁 Structure des Dossiers Cloudinary

```
pumppro/
├── avatars/           # Avatars utilisateurs
│   └── user_id.jpg
├── challenges/        # Images de challenges
│   └── challenge_id.jpg
└── bug_reports/       # Screenshots de bug reports
    └── report_id.jpg
```

## 🔄 Migration des Images Existantes

Si vous avez déjà des images avec l'ancien système, voici comment les migrer:

### Script SQL pour identifier les images locales

```sql
-- Compter les avatars à migrer
SELECT COUNT(*)
FROM users
WHERE avatar LIKE '%pumppro-backend.onrender.com%'
   OR avatar LIKE '%pompeurpro.com%';

-- Lister les utilisateurs concernés
SELECT id, name, avatar
FROM users
WHERE avatar LIKE '%pumppro-backend.onrender.com%'
   OR avatar LIKE '%pompeurpro.com%';
```

### Options de Migration

**Option 1: Réupload manuel**
- Demandez aux utilisateurs de réuploader leur avatar

**Option 2: Migration automatique** (nécessite un script)
- Télécharger les images depuis les anciennes URLs
- Les réuploader vers Cloudinary
- Mettre à jour les URLs dans la DB

## 💰 Limites du Plan Gratuit

- ✅ **25 GB** de stockage
- ✅ **25 GB** de bande passante/mois
- ✅ **25 000** transformations/mois
- ✅ CDN inclus

Pour PumpPro, cela représente environ:
- **~50,000 avatars** (si 500KB chacun)
- Largement suffisant pour commencer!

## 🔐 Sécurité

**Ne partagez JAMAIS:**
- ❌ Votre `CLOUDINARY_API_SECRET`
- ❌ Vos credentials dans le code
- ❌ Les credentials dans Git

**Toujours:**
- ✅ Utilisez des variables d'environnement
- ✅ Ajoutez `.env` au `.gitignore`
- ✅ Utilisez des variables d'environnement Render

## 📚 Ressources

- [Documentation Cloudinary](https://cloudinary.com/documentation)
- [Go SDK Documentation](https://github.com/cloudinary/cloudinary-go)
- [Image Transformations](https://cloudinary.com/documentation/image_transformations)
- [Dashboard Cloudinary](https://cloudinary.com/console)

## ❓ FAQ

**Q: Les anciennes images vont-elles disparaître?**
R: Oui, les images stockées localement sur Render sont perdues à chaque redéploiement. Configurez Cloudinary dès maintenant!

**Q: Que se passe-t-il si je ne configure pas Cloudinary?**
R: Le système utilisera le stockage local (éphémère). Les images seront perdues lors des redéploiements.

**Q: Puis-je utiliser un autre service (AWS S3, Supabase)?**
R: Oui, mais cela nécessite de modifier le code. Cloudinary est le plus simple et gratuit.

**Q: Comment désactiver le stockage local?**
R: Une fois Cloudinary configuré, toutes les nouvelles images iront automatiquement vers Cloudinary.

## ✨ Fonctionnalités Supplémentaires

Le service Cloudinary inclut également:

### Upload de Challenges
```go
cloudinaryService.UploadChallengeImage(ctx, file, challengeID)
```

### Upload de Screenshots de Bugs
```go
cloudinaryService.UploadBugReportScreenshot(ctx, file, reportID)
```

### URLs Optimisées
```go
cloudinaryService.GetOptimizedURL(publicID, 300, 300)
```

### Suppression d'Images
```go
cloudinaryService.DeleteImage(ctx, publicID)
```

---

**✅ Prêt à Déployer!**

Une fois configuré, vos images seront:
- 🌍 Accessibles partout dans le monde via CDN
- ⚡ Optimisées automatiquement
- 🔒 Sécurisées avec HTTPS
- 💾 Persistantes à vie (pas de perte au redéploiement)
