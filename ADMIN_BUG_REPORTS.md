# Routes Admin - Gestion des Bug Reports

## 📋 Vue d'ensemble

Les administrateurs disposent de routes spécialisées pour gérer efficacement les bug reports soumis par les utilisateurs.

## 🔐 Authentification

Toutes ces routes nécessitent:
- ✅ Authentification (token JWT)
- ✅ Privilèges admin (`is_admin = true`)

## 📡 Routes Disponibles

### 1. Liste des Bug Reports (avec filtrage avancé)

**Endpoint:** `GET /admin/bug-reports`

**Query Parameters:**
- `status` - Filtrer par statut: `pending`, `in_progress`, `resolved`, `closed`
- `severity` - Filtrer par gravité: `low`, `medium`, `high`, `critical`
- `category` - Filtrer par catégorie: `bug`, `feature`, `ui`, `performance`, `crash`, `other`
- `search` - Recherche dans le titre et la description
- `sort` - Trier par: `created_at`, `updated_at`, `severity` (défaut: `created_at`)
- `order` - Ordre: `asc`, `desc` (défaut: `desc`)
- `limit` - Nombre de résultats par page (défaut: 50)
- `offset` - Nombre de résultats à ignorer (pagination)

**Exemple de Requête:**
```bash
GET /admin/bug-reports?status=pending&severity=high&limit=20&offset=0
```

**Réponse:**
```json
{
  "success": true,
  "data": {
    "reports": [
      {
        "id": "uuid",
        "userId": "user-uuid",
        "title": "App crashes on logout",
        "description": "When I press logout...",
        "category": "crash",
        "severity": "high",
        "status": "pending",
        "deviceInfo": {...},
        "appVersion": "1.2.3",
        "pageUrl": "/settings",
        "errorStack": "Error: ...",
        "screenshotUrl": "https://...",
        "userEmail": "user@example.com",
        "createdAt": "2025-10-31T10:00:00Z",
        "updatedAt": "2025-10-31T10:00:00Z",
        "resolvedAt": null,
        "resolvedBy": null,
        "adminNotes": null,
        "userName": "John Doe",
        "userAvatar": "https://..."
      }
    ],
    "pagination": {
      "total": 45,
      "limit": 20,
      "offset": 0,
      "count": 20
    },
    "filters": {
      "status": "pending",
      "severity": "high",
      "category": "",
      "search": ""
    }
  }
}
```

### 2. Résoudre un Bug Report

**Endpoint:** `POST /admin/bug-reports/{reportId}/resolve`

**Body (optionnel):**
```json
{
  "adminNotes": "Fixed in version 1.2.4"
}
```

**Effet:**
- ✅ Change le statut à `resolved`
- ✅ Ajoute `resolved_at` = maintenant
- ✅ Ajoute `resolved_by` = ID de l'admin
- ✅ Ajoute les notes admin si fournies

**Exemple:**
```bash
curl -X POST https://pumppro-backend.onrender.com/admin/bug-reports/123/resolve \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"adminNotes": "Fixed in v1.2.4"}'
```

**Réponse:**
```json
{
  "success": true,
  "data": {
    "id": "123",
    "status": "resolved",
    "resolvedAt": "2025-10-31T10:30:00Z",
    "resolvedBy": "admin-uuid",
    "adminNotes": "Fixed in v1.2.4",
    ...
  }
}
```

### 3. Assigner un Bug Report à un Admin

**Endpoint:** `POST /admin/bug-reports/{reportId}/assign`

**Body:**
```json
{
  "adminId": "admin-uuid",
  "notes": "John will investigate this issue"
}
```

**Effet:**
- ✅ Change le statut à `in_progress`
- ✅ Assigne à l'admin spécifié (`resolved_by` = adminId)
- ✅ Ajoute les notes si fournies

**Exemple:**
```bash
curl -X POST https://pumppro-backend.onrender.com/admin/bug-reports/123/assign \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "adminId": "admin-uuid-here",
    "notes": "Assigned to John for investigation"
  }'
```

**Réponse:**
```json
{
  "success": true,
  "data": {
    "id": "123",
    "status": "in_progress",
    "resolvedBy": "admin-uuid-here",
    "adminNotes": "Assigned to John for investigation",
    ...
  }
}
```

## 📊 Workflow de Gestion

### Workflow Typique

```
1. Bug Report Créé
   ↓ status: "pending"

2. Admin Consulte la Liste
   GET /admin/bug-reports?status=pending

3. Admin Assigne à Lui-même ou Collègue
   POST /admin/bug-reports/{id}/assign
   ↓ status: "in_progress"

4. Admin Travaille sur le Fix
   (développement, tests)

5. Admin Résout le Bug
   POST /admin/bug-reports/{id}/resolve
   ↓ status: "resolved"
```

### Filtrage par Priorité

**Bugs Critiques:**
```bash
GET /admin/bug-reports?severity=critical&status=pending&sort=created_at&order=asc
```

**Bugs Non Résolus:**
```bash
GET /admin/bug-reports?status=pending,in_progress
```

**Bugs Résolus Aujourd'hui:**
```bash
GET /admin/bug-reports?status=resolved&sort=resolved_at&order=desc
```

## 🔍 Cas d'Usage

### Dashboard Admin - Vue d'Ensemble

```javascript
// Récupérer les statistiques
const stats = {
  pending: await fetch('/admin/bug-reports?status=pending').then(r => r.json()),
  critical: await fetch('/admin/bug-reports?severity=critical&status=pending').then(r => r.json()),
  inProgress: await fetch('/admin/bug-reports?status=in_progress').then(r => r.json())
}
```

### Recherche de Bugs

```bash
# Rechercher "crash" dans le titre ou description
GET /admin/bug-reports?search=crash

# Bugs de performance
GET /admin/bug-reports?category=performance

# Bugs iOS
GET /admin/bug-reports?search=iOS
```

### Assignation Automatique

```javascript
// Assigner automatiquement les bugs critiques à l'admin de garde
const criticalBugs = await fetch('/admin/bug-reports?severity=critical&status=pending')
  .then(r => r.json())

for (const bug of criticalBugs.data.reports) {
  await fetch(`/admin/bug-reports/${bug.id}/assign`, {
    method: 'POST',
    body: JSON.stringify({
      adminId: 'admin-on-duty-id',
      notes: 'Auto-assigned: Critical bug'
    })
  })
}
```

## 📈 Métriques Disponibles

Via les filtres, vous pouvez obtenir:

- **Total de bugs en attente**: `?status=pending`
- **Bugs critiques non résolus**: `?severity=critical&status=pending`
- **Bugs assignés à un admin**: Filtrer par `resolvedBy` dans la réponse
- **Bugs par catégorie**: `?category=crash`
- **Tendance de résolution**: Comparer les périodes avec `sort=resolved_at`

## 🎯 Routes Existantes (Non Admin)

Ces routes existent aussi pour tous les utilisateurs:

- `POST /bug-reports` - Créer un bug report
- `GET /bug-reports` - Lister les bug reports (publics)
- `GET /bug-reports/{id}` - Voir un bug report
- `PUT/PATCH /bug-reports/{id}` - Modifier (propriétaire ou admin)
- `DELETE /bug-reports/{id}` - Supprimer (propriétaire ou admin)
- `GET /bug-reports/stats` - Statistiques globales

## ⚡ Différences Routes Admin vs Routes Publiques

| Fonctionnalité | Routes Publiques | Routes Admin |
|---|---|---|
| **Filtrage avancé** | ❌ Limité | ✅ Complet (statut, sévérité, catégorie) |
| **Voir tous les bugs** | ❌ Non | ✅ Oui |
| **Info utilisateur** | ❌ Non | ✅ Oui (nom, avatar) |
| **Assignation** | ❌ Non | ✅ Oui |
| **Résolution rapide** | ❌ Non | ✅ Oui (1 clic) |
| **Tri personnalisé** | ❌ Non | ✅ Oui |
| **Recherche** | ❌ Non | ✅ Oui |

## 🔧 Intégration Frontend

### React/Next.js Example

```typescript
// services/admin/bugReports.ts
export const adminBugReports = {
  list: async (filters?: {
    status?: string
    severity?: string
    category?: string
    search?: string
    sort?: string
    order?: string
    limit?: number
    offset?: number
  }) => {
    const params = new URLSearchParams(filters as any)
    return fetch(`/admin/bug-reports?${params}`)
      .then(r => r.json())
  },

  resolve: async (reportId: string, adminNotes?: string) => {
    return fetch(`/admin/bug-reports/${reportId}/resolve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ adminNotes })
    }).then(r => r.json())
  },

  assign: async (reportId: string, adminId: string, notes?: string) => {
    return fetch(`/admin/bug-reports/${reportId}/assign`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ adminId, notes })
    }).then(r => r.json())
  }
}
```

## 📝 Notes

- Les bug reports incluent maintenant le nom et l'avatar de l'utilisateur pour faciliter le contact
- Le champ `resolvedBy` contient l'ID de l'admin qui a résolu ou qui est assigné
- Les `adminNotes` sont visibles uniquement par les admins
- La pagination permet de gérer efficacement un grand nombre de bug reports

---

**Créé le:** 31 Octobre 2025
**Version:** 1.0.0
