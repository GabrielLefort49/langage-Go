# TP Final - Mira

Ce dépôt regroupe les cinq TP (TP1..TP5) en une version intégrée :

- `TP1` : CLI cliente — [vidéo de preuve](TP1/Video%20preuve%20TP1.mp4)
- `TP2` : API HTTP principale (création / lecture / mise à jour / suppression) — [README](TP2/README.md)
- `TP3` : exercices pédagogiques (concurrence) — [README](TP3/README.md)
- `TP4` : API + enricher asynchrone + recherche — [README](TP4/README.md) · [vidéo de preuve](TP4/Vid%C3%A9o%20preuve%20TP4.mp4)
- `TP5` : serveur MCP (`mira-mcp`) exposant la mémoire aux agents IA — [README](TP5/README.md)

Un README détaillé est fourni pour chaque TP sauf `TP1` (CLI simple, voir sa
vidéo de preuve). Des vidéos de preuve sont disponibles pour `TP1` et `TP4`.

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


Preparation des documents non fini 