# TP Final - Mira

Ce dépôt regroupe les quatre TP (TP1..TP4) en une version intégrée :

- `TP1` : CLI cliente
- `TP2` : API HTTP principale (création / lecture / mise à jour / suppression)
- `TP3` : exercices pédagogiques (concurrence)
- `TP4` : API + enricher asynchrone + recherche

Tout le système partage une base PostgreSQL (variable `DATABASE_URL`).

**Ports par défaut (modifiable via `PORT`)**
- TP4 : `http://localhost:8084` (API + enricher)
- TP2 : `http://localhost:8085` (API principale)
- Swagger aggregator (UI statique) : `http://localhost:8090`

## Pré-requis

- Go installé
- Docker (optionnel pour Postgres) ou PostgreSQL local

## Démarrage rapide (recommended)

1) Démarrer Postgres (Docker)

```powershell
docker run --name mira-postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15
docker exec -it mira-postgres psql -U postgres -c "CREATE DATABASE mira;"
```

2) Lancer `TP4` (enricher + API)

```powershell
cd TP-final\TP4
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/mira?sslmode=disable"
$env:PORT = "8084"
go run ./cmd/api
```

3) Lancer `TP2` (API principale)

```powershell
cd TP-final\TP2
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/mira?sslmode=disable"
$env:PORT = "8085"
go run ./cmd/api
```

4) Lancer l’UI Swagger (agrégateur statique)

```powershell
cd TP-final\swagger
$env:PORT = "8090"
go run ./server.go
# ouvrir http://localhost:8090 dans le navigateur
```

5) (Optionnel) Utiliser la CLI `TP1`

```powershell
cd TP-final\TP1
$env:MIRA_API = "http://localhost:8085"
go run . add "Titre" "Contenu de test"
```

## Tester via Swagger

- Ouvrez `http://localhost:8090`.
- Collez l’URL du spec OpenAPI d’un service (par ex. `http://localhost:8085/openapi.yaml` pour TP2 ou `http://localhost:8084/swagger.json` pour TP4).

Remarque : j’ai activé CORS côté TP2 et TP4 afin que l’agrégateur Swagger puisse charger les specs cross-origin.

## Commandes utiles

- Lister les notes (TP2) : `GET http://localhost:8085/api/v1/notes`
- Créer une note (TP2) : `POST http://localhost:8085/api/v1/notes` (JSON `{title,content}`)
- Rechercher (TP4) : `GET http://localhost:8084/api/v1/search?q=...`

## Débogage rapide

- Vérifier qu’un port est libre : `netstat -ano | findstr :8084`
- Tuer un processus : `taskkill /PID <pid> /F`

Si vous voulez, je peux committer ce `README.md` mis à jour et l’ouvrir dans l’éditeur.
