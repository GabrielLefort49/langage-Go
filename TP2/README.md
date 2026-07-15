# Mira — API v1

API HTTP de notes, avec stockage en mémoire (une `map` protégée par `sync.RWMutex`). Elle nécessite Go 1.26 ou plus récent.

```sh
go run ./cmd/api
```

Le serveur écoute sur `http://localhost:8080`.

## Swagger

Après le démarrage du serveur, ouvre [http://localhost:8080/swagger/](http://localhost:8080/swagger/) dans ton navigateur. Cette interface permet de tester toutes les routes sans Postman.

Pour créer une note depuis Swagger :

1. Déplie `POST /api/v1/notes`.
2. Clique sur **Try it out**.
3. Remplace le JSON proposé, par exemple :

   ```json
   {
     "title": "Ma première note",
     "content": "Contenu créé depuis Swagger"
   }
   ```

4. Clique sur **Execute**. La réponse doit avoir le statut `201` et contient l'identifiant de la note, par exemple `note-1`.

La spécification OpenAPI est aussi disponible sur [http://localhost:8080/openapi.yaml](http://localhost:8080/openapi.yaml). Elle peut être importée dans Postman ou Swagger Editor.

### Génération de la spécification (bonus)

Le fichier [swagger.yaml](internal/http/openapi/generated/swagger.yaml) est généré avec l'outil `swag` à partir des annotations des handlers. Pour le régénérer après une modification des routes :

```powershell
& "$(go env GOPATH)\bin\swag.exe" init --generalInfo cmd/api/main.go --parseInternal --output internal/http/openapi/generated --outputTypes yaml
```

La version générée est également servie par l'API sur [http://localhost:8080/swagger-generated.yaml](http://localhost:8080/swagger-generated.yaml).

Swagger UI charge ses fichiers d'interface depuis un CDN : le navigateur doit donc avoir accès à Internet.

## Routes

| Méthode | Route | Description |
| --- | --- | --- |
| POST | `/api/v1/notes` | Crée une note |
| GET | `/api/v1/notes?limit=20&offset=0` | Liste les notes (pagination bonus) |
| GET | `/api/v1/notes/{id}` | Récupère une note |
| PATCH | `/api/v1/notes/{id}` | Met à jour partiellement une note |
| DELETE | `/api/v1/notes/{id}` | Supprime une note |
| GET | `/api/v1/search?q=texte` | Recherche dans le titre et le contenu |

Les payloads de création et modification sont des objets JSON. `title` est obligatoire à la création, non vide et limité à 200 caractères ; `content` est limité à 10 000 caractères.

```sh
curl -i -X POST http://localhost:8080/api/v1/notes ^
  -H "Content-Type: application/json" ^
  -d "{\"title\":\"Courses\",\"content\":\"Acheter du lait\"}"

curl http://localhost:8080/api/v1/notes?limit=10^&offset=0
curl http://localhost:8080/api/v1/notes/note-1
curl -X PATCH http://localhost:8080/api/v1/notes/note-1 -H "Content-Type: application/json" -d "{\"content\":\"Acheter du lait et du pain\"}"
curl -X DELETE -i http://localhost:8080/api/v1/notes/note-1
curl "http://localhost:8080/api/v1/search?q=lait"
```

## Réponses et erreurs

Les succès utilisent l'enveloppe `{ "data": ... }`. Les listes ajoutent `meta` (`total`, `limit`, `offset`). Les erreurs utilisent systématiquement :

```json
{"error":{"code":"bad_request","message":"title is required"}}
```

Codes possibles : `201` (création), `200` (lecture, modification, recherche), `204` (suppression), `400` (payload, paramètres ou recherche invalides), `404` (note absente), `405` (méthode non autorisée), `500` (panic récupérée), `503` (timeout).

Les middlewares ajoutent/propagent `X-Request-ID`, journalisent chaque requête avec `slog`, récupèrent les panics et imposent un délai de 5 secondes.

## Tests

```sh
go test ./...
```
