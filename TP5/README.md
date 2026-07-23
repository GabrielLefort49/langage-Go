# TP5 — Serveur MCP `mira-mcp`

Expose la mémoire de **mira** à un agent IA (Claude Code, Claude Desktop, IDE…)
via le **Model Context Protocol (MCP)**. L'agent peut alors **chercher, lire et
créer des notes pendant une conversation**.

Le serveur parle **JSON-RPC 2.0 sur le transport stdio** et s'appuie sur le SDK
officiel [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

> ⚠️ **Le serveur passe toujours par l'API HTTP de mira (TP4), jamais par la base
> en direct.** C'est ce qui garantit que l'enrichissement automatique (tags,
> résumé, embedding) soit déclenché pour chaque note créée.

---

## Outils exposés

| Tool | Paramètres | Rôle |
| --- | --- | --- |
| `search_notes` | `query` (string, requis), `limit` (int, défaut 10) | Recherche hybride full-text + vectorielle |
| `get_note` | `id` (string, requis) | Retourne une note complète (contenu, tags, résumé, statut) |
| `add_note` | `title`, `content` (requis), `tags` (optionnel) | Crée une note (et attend brièvement son enrichissement) |
| `list_recent_notes` | `limit` (int, défaut 10) | Dernières notes créées |

Chaque outil déclare une **description soignée** et un **schéma JSON strict**
(`additionalProperties: false`, champs `required`, `minLength`, valeurs par
défaut). L'agent choisit ses outils en lisant ces descriptions : elles font
partie du contrat.

### Détails de comportement

- **Validation** : les entrées sont validées contre le schéma JSON *avant*
  d'atteindre le handler ; les erreurs sont renvoyées comme erreurs MCP propres
  (jamais de `panic`, jamais de stack trace brute côté client).
- **Timeout** : chaque appel à l'API sous-jacente est borné par un `context`
  (`MIRA_TIMEOUT`, 15 s par défaut).
- **`add_note` et l'enrichissement** : après création, l'outil interroge
  `get_note` en boucle courte jusqu'à `enrichment_status: done` (au plus
  `MIRA_ENRICH_WAIT`, 5 s par défaut) puis renvoie la note enrichie. Si
  l'enrichissement n'est pas terminé à temps, la note `pending` est renvoyée
  avec un message invitant à rappeler `get_note`.
- **Tags fournis par l'utilisateur** : l'endpoint de création de mira n'accepte
  que `{title, content}`. Les `tags` optionnels sont donc repliés dans le corps
  de la note (`\n\nTags: …`) afin d'être persistés et restés cherchables ; mira
  génère par ailleurs ses propres tags lors de l'enrichissement.
- **Logs** : aucun log sur **stdout** (réservé au protocole). Tout passe par
  `slog` sur **stderr** (`MIRA_LOG_LEVEL` = `debug|info|warn|error`).

---

## Configuration (variables d'environnement)

| Variable | Défaut | Rôle |
| --- | --- | --- |
| `MIRA_API_URL` | `http://localhost:8084` | Base URL de l'API mira (TP4) |
| `MIRA_TIMEOUT` | `15s` | Timeout par requête HTTP |
| `MIRA_ENRICH_WAIT` | `5s` | Attente max de l'enrichissement dans `add_note` |
| `MIRA_LOG_LEVEL` | `info` | Niveau de log (stderr) |

---

## Pré-requis

L'API mira **TP4** (API + enricher) doit tourner et être joignable :

```powershell
# 1) Postgres (Docker)
docker run --name mira-postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15
docker exec -it mira-postgres psql -U postgres -c "CREATE DATABASE mira;"

# 2) API mira TP4 (applique les migrations et démarre l'API + enricher)
cd TP-final\TP4
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/mira?sslmode=disable"
$env:PORT = "8084"
go run ./cmd/api
# -> http://localhost:8084  (GET /healthz doit répondre "ok")
```

---

## Installation

```powershell
cd TP-final\TP5
go install ./cmd/mira-mcp
```

Le binaire `mira-mcp` est installé dans `$(go env GOPATH)\bin`
(par ex. `C:\Users\<vous>\go\bin\mira-mcp.exe`). Assurez-vous que ce dossier est
dans votre `PATH`, ou utilisez le chemin absolu du binaire dans les configs
ci-dessous.

> Alternative sans installer : `go run ./cmd/mira-mcp` depuis `TP-final/TP5`.

---

## Démarrage rapide (checklist)

1. Postgres up + DB créée (section **Pré-requis**)
2. `go run ./cmd/api` (TP4) → `GET /healthz` répond `ok`
3. `go install ./cmd/mira-mcp` (TP5) → `mira-mcp` accessible dans le `PATH`
4. `.mcp.json` présent à la racine de `TP-final/TP5` (fourni dans ce dossier)
5. Ouvrir Claude Code depuis `TP-final/TP5` (ou copier `.mcp.json` à la racine
   du projet ouvert) → `/mcp` doit lister `mira` avec 4 tools

```powershell
cd TP-final\TP5
go build ./...
go vet ./...
go test ./...
```
Les trois doivent passer sans erreur : c'est la validation la plus fiable
(le test manuel du transport stdio ci-dessous est utile mais plus fragile,
voir la note associée).

---

## Enregistrement dans **Claude Code**

Un fichier [`.mcp.json`](.mcp.json) est fourni à la racine de ce dossier. Deux méthodes :

### A. Fichier `.mcp.json` (scope projet)

Copiez `.mcp.json` à la racine du projet où vous lancez Claude Code (ou utilisez
celui de ce dossier). Claude Code le découvre automatiquement :

```json
{
  "mcpServers": {
    "mira": {
      "command": "mira-mcp",
      "args": [],
      "env": {
        "MIRA_API_URL": "http://localhost:8084",
        "MIRA_TIMEOUT": "15s",
        "MIRA_ENRICH_WAIT": "5s",
        "MIRA_LOG_LEVEL": "info"
      }
    }
  }
}
```

> Si `mira-mcp` n'est pas dans le `PATH`, remplacez `"command"` par le chemin
> absolu du binaire, ou utilisez `"command": "go", "args": ["run", "./cmd/mira-mcp"]`
> en lançant Claude Code depuis `TP-final/TP5`.

### B. En une commande (CLI)

```powershell
claude mcp add mira --env MIRA_API_URL=http://localhost:8084 -- mira-mcp
```

Vérifiez ensuite :

```powershell
claude mcp list        # doit afficher "mira"
```

Dans Claude Code, `/mcp` liste les serveurs connectés et leurs outils.

---

## Enregistrement dans **Claude Desktop**

Éditez le fichier de config de Claude Desktop :

- Windows : `%APPDATA%\Claude\claude_desktop_config.json`
- macOS : `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "mira": {
      "command": "C:\\Users\\<vous>\\go\\bin\\mira-mcp.exe",
      "env": { "MIRA_API_URL": "http://localhost:8084" }
    }
  }
}
```

Redémarrez Claude Desktop ; l'icône outils doit exposer les 4 tools `mira`.

---

## Exemples de prompts

Une fois le serveur enregistré et l'API TP4 démarrée, demandez à l'agent :

- « **Retrouve ma note sur les channels Go** et résume-la. »
  → l'agent appelle `search_notes` puis `get_note`.
- « **Ajoute une note** intitulée *"Réunion sprint"* avec ce compte-rendu : … »
  → l'agent appelle `add_note` ; la note revient avec `enrichment_status: done`.
- « **Retrouve ma note sur les channels Go et ajoute une note résumant ce qu'on
  vient de faire.** »
  → enchaîne `search_notes` + `add_note`.
- « **Montre-moi mes dernières notes.** »
  → `list_recent_notes`.

---

## Développement & tests

```powershell
cd TP-final\TP5
go build ./...                 # compilation
go vet ./...                   # analyse statique
go test ./...                  # tests (API mira simulée en mémoire)
```

Les tests (`internal/mcpserver/server_test.go`) branchent un client MCP en
mémoire sur le serveur, avec une **fausse API mira** reproduisant la transition
`pending → done` de l'enrichissement. Ils couvrent : listing des outils,
schémas non vides, flux complet `add_note → search_notes → get_note →
list_recent_notes`, et validation des entrées.

### Vérifier manuellement le transport stdio

Le serveur ne doit rien écrire sur stdout à part le protocole :

```powershell
'{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"x","version":"1"}}}' , `
'{"jsonrpc":"2.0","method":"notifications/initialized"}' , `
'{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | mira-mcp
# -> stdout : réponses JSON-RPC (initialize + 4 tools)
# -> stderr : logs slog
```

> **Note** : ce test manuel est fragile sous Windows/Git Bash — dès que le
> pipe stdin atteint EOF (fin d'entrée), le serveur considère la connexion
> fermée (`server is closing: EOF`) et peut s'arrêter avant d'avoir eu le
> temps d'écrire les réponses sur stdout, même si tout fonctionne
> correctement côté protocole. C'est une limite du test, pas un bug du
> serveur. Pour une vérification fiable, préférez `go test ./...`
> (client MCP en mémoire qui garde la connexion ouverte) ou l'intégration
> réelle dans Claude Code (`/mcp`).

---

## Architecture

```
Agent IA (Claude Code / Desktop)
        │  JSON-RPC 2.0 (stdio)
        ▼
   mira-mcp  ── internal/mcpserver ─ 4 tools + schémas stricts
        │  HTTP (internal/mira.Client, context timeout)
        ▼
   API mira TP4  (POST/GET /api/v1/notes, GET /api/v1/search)
        │
        ▼
   Enricher async  →  tags, résumé, embedding  →  enrichment_status: done
```

- `cmd/mira-mcp` : point d'entrée, configuration via env, logger stderr,
  démarrage du transport stdio, arrêt propre sur SIGINT/SIGTERM.
- `internal/mcpserver` : définition des outils, schémas JSON, handlers,
  attente d'enrichissement, mise en forme des réponses.
- `internal/mira` : client HTTP minimal de l'API mira (enveloppe `data`/`error`,
  erreurs typées, timeouts).
