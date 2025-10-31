# Problème de Stockage sur Render.com

## Problème Actuel

Sur Render.com, le système de fichiers est **éphémère**. Cela signifie que:
- Les fichiers uploadés (avatars, photos, etc.) sont stockés localement dans `/uploads`
- Ces fichiers sont **perdus à chaque redéploiement** de l'application
- Les URLs dans la base de données pointent vers des fichiers qui n'existent plus

## Symptômes

1. **Route `/admin/photos`** retourne un tableau vide car les fichiers n'existent plus localement
2. **Route `/avatars/{filename}`** retourne "image non trouvée" car le fichier a été supprimé lors du redéploiement
3. Les utilisateurs perdent leurs avatars après chaque mise à jour du backend

## Solutions Recommandées

### Solution 1: Cloudinary (Recommandé - GRATUIT jusqu'à 25GB)

```go
// Installation
go get github.com/cloudinary/cloudinary-go/v2

// Configuration dans .env
CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=your_api_key
CLOUDINARY_API_SECRET=your_api_secret

// Exemple d'upload
import "github.com/cloudinary/cloudinary-go/v2"

func uploadToCloudinary(file multipart.File, filename string) (string, error) {
    cld, _ := cloudinary.NewFromParams(
        os.Getenv("CLOUDINARY_CLOUD_NAME"),
        os.Getenv("CLOUDINARY_API_KEY"),
        os.Getenv("CLOUDINARY_API_SECRET"),
    )

    uploadResult, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
        PublicID: filename,
        Folder:   "avatars",
    })

    return uploadResult.SecureURL, err
}
```

**Avantages:**
- Gratuit jusqu'à 25GB de stockage
- CDN global pour des temps de chargement rapides
- Optimisation d'images automatique
- Transformation d'images (resize, crop, etc.)

### Solution 2: AWS S3

```go
// Installation
go get github.com/aws/aws-sdk-go/aws
go get github.com/aws/aws-sdk-go/service/s3

// Configuration
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=us-east-1
AWS_BUCKET_NAME=your_bucket_name
```

**Avantages:**
- Très fiable et scalable
- Intégration avec CloudFront pour CDN
- Tarification à l'usage

**Inconvénients:**
- Coût (environ $0.023 par GB par mois)
- Configuration plus complexe

### Solution 3: Supabase Storage

```go
// Utiliser l'API REST de Supabase
// Documentation: https://supabase.com/docs/guides/storage
```

**Avantages:**
- Gratuit jusqu'à 1GB
- Facile à intégrer
- S'intègre bien avec PostgreSQL

### Solution 4: Render Disks (PAS GRATUIT)

Render propose des volumes persistants, mais:
- Coût: $0.25 par GB par mois minimum
- Minimum 1GB = $0.25/mois supplémentaire

## Migration des URLs Existantes

Actuellement, les avatars dans la DB utilisent des URLs complètes avec d'anciens domaines:
```sql
SELECT avatar FROM users WHERE avatar IS NOT NULL;
-- Résultat: https://api.pompeurpro.com/avatars/user_001.jpg
```

### Script de Migration (à exécuter après l'implémentation d'un service cloud)

```sql
-- 1. Identifier les avatars avec des URLs externes
SELECT id, name, avatar
FROM users
WHERE avatar LIKE 'http%';

-- 2. Après upload vers Cloudinary/S3, mettre à jour les URLs
UPDATE users
SET avatar = 'https://res.cloudinary.com/your-cloud/image/upload/avatars/user_001.jpg'
WHERE id = 'user-id';
```

## Correctifs Temporaires Appliqués

1. **GetAvatar** - Extrait le nom du fichier de l'URL complète
2. **GetAllPhotos** - Utilise `LENGTH(avatar) > 0` au lieu de `avatar != ''`
3. Les fichiers locaux sont créés temporairement mais seront perdus au redéploiement

## Recommandation

**Utilisez Cloudinary** pour:
- Plan gratuit généreux (25GB)
- CDN global intégré
- Optimisation d'images automatique
- Pas de configuration complexe
- Parfait pour les applications en phase de démarrage

## Étapes d'Implémentation avec Cloudinary

1. Créer un compte sur https://cloudinary.com
2. Récupérer les credentials (Cloud Name, API Key, API Secret)
3. Installer le package Go: `go get github.com/cloudinary/cloudinary-go/v2`
4. Modifier `UploadAvatar` pour uploader vers Cloudinary
5. Migrer les avatars existants vers Cloudinary
6. Mettre à jour les URLs dans la base de données
