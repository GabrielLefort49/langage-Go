# TP4 - Mira avec PostgreSQL, enrichissement asynchrone et recherche avancée

Cette version du TP4 remplace le stockage mémoire par PostgreSQL, ajoute un enrichissement asynchrone et expose une API HTTP de test.

## Fonctionnalités livrées

- Stockage persistant avec PostgreSQL via `pgx`
- Schéma SQL avec `notes`, `note_tags`, `note_embeddings`
- Enrichissement automatique déclenché à chaque création / modification de note
- Queue interne + workers bornés
- Statut d’enrichissement `pending`, `done`, `failed`
- Recherche simple via le endpoint `/api/v1/search`
- Endpoint de santé `/healthz`
- Swagger minimal sur `/swagger` et `/swagger.json`
- CLI qui appelle l’API au lieu du fichier JSONL local

## Prérequis

- Go 1.20+
- Docker (recommandé) ou PostgreSQL local

## 1. Démarrer PostgreSQL

### Option A — Docker (recommandée)

```powershell
docker run --name mira-pg -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15
```

Créer la base `mira` :

```powershell
docker exec -it mira-pg psql -U postgres -c "CREATE DATABASE mira;"
```

### Option B — PostgreSQL local

Créer la base `mira` avec `psql` :

```powershell
psql -U postgres -c "CREATE DATABASE mira;"
```

## 2. Installer les dépendances Go

Depuis le dossier TP4 :

```powershell
cd C:\Users\gabri\Documents\Go\mira\TP4
go mod tidy
```

## 3. Lancer l’API

```powershell
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/mira?sslmode=disable"
$env:PORT="8081"
go run ./cmd/api
```

Le serveur écoute sur `http://localhost:8081`.

## 4. Tester l’API HTTP

### Santé

```powershell
Invoke-RestMethod -Uri http://localhost:8081/healthz
```

### Créer une note

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8081/api/v1/notes -Body (@{title='Test TP4';content='Contenu de test pour enrichissement automatique.'} | ConvertTo-Json) -ContentType 'application/json'
```

### Récupérer une note

Remplace `<ID>` par l’identifiant retourné par la création.

```powershell
Invoke-RestMethod -Uri "http://localhost:8081/api/v1/notes/<ID>"
```

### Mettre à jour une note

```powershell
Invoke-RestMethod -Method Patch -Uri "http://localhost:8081/api/v1/notes/<ID>" -Body (@{content='Contenu mis à jour pour vérifier l’enrichissement.'} | ConvertTo-Json) -ContentType 'application/json'
```

### Rechercher des notes

```powershell
Invoke-RestMethod -Uri "http://localhost:8081/api/v1/search?q=test"
```

### Lister les notes

```powershell
Invoke-RestMethod -Uri "http://localhost:8081/api/v1/notes?limit=20&offset=0"
```

## 5. Swagger

- UI: http://localhost:8081/swagger
- Spec: http://localhost:8081/swagger.json

## 6. CLI

La CLI du TP1 a été adaptée pour appeler l’API au lieu du fichier JSONL.

Exemples :

```powershell
cd C:\Users\gabri\Documents\Go\mira
go run .\TP1\main.go add "Titre" "Contenu"
go run .\TP1\main.go list
go run .\TP1\main.go search "test"
```

## Correctifs appliqués

- suppression de la dépendance à `pgvector` pour que le TP4 fonctionne avec PostgreSQL standard
- migration SQL compatible avec un conteneur PostgreSQL simple
- ajout d’un endpoint `/healthz`
- ajout d’un endpoint `/swagger` et `/swagger.json`
- ajout de la route `/api/v1/search`
- état `enrichment_status` géré en `pending`, `done` ou `failed`
