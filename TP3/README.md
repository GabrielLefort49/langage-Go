# TP #3 — Concurrence et goroutines en Go

Ce dépôt contient les solutions du TP sur les goroutines, les channels, les worker pools et les race conditions en Go.

Chaque exercice est placé dans son propre dossier. Cela permet à chaque programme de posséder sa fonction `main` et d'être exécuté indépendamment.

## Prérequis

- Go installé (vérification avec `go version`)
- Un terminal ouvert à la racine du projet

```powershell
cd C:\Users\gabri\Documents\Go\mira\TP3
go version
```

Le module Go utilisé par le projet est défini dans `go.mod`.

## Structure du projet

```text
TP3/
├── ex1/      # Première goroutine
├── ex2/      # Synchronisation avec WaitGroup
├── ex3/      # Somme parallèle avec channel
├── ex4/      # Worker pool
├── ex5/      # select et timeout
├── ex6/      # Race condition et Mutex
├── bonus/    # Annulation avec context.Context
├── go.mod
└── README.md
```

## Commandes utiles

Exécuter un exercice :

```powershell
go run ./ex1
```

Remplacer `ex1` par le dossier souhaité : `ex2`, `ex3`, `ex4`, `ex5`, `ex6` ou `bonus`.

Les exercices 3, 4, 5 et le bonus sont interactifs : saisir les valeurs demandées, puis appuyer sur Entrée. Par exemple :

```text
Nombre de workers : 4
Nombre de jobs : 20
```

Formater tous les fichiers Go :

```powershell
gofmt -w ex1/main.go ex2/main.go ex3/main.go ex4/main.go ex5/main.go ex6/main.go bonus/main.go
```

Vérifier que tous les programmes compilent :

```powershell
go test ./...
```

> Il n'y a pas de tests unitaires dans ce TP. Cette commande compile toutefois tous les packages et détecte les erreurs de compilation.

---

## Exercice 1 — Première goroutine

Fichier : `ex1/main.go`

```powershell
go run ./ex1
```

Le programme contient deux fonctions :

- `afficherLettres()` affiche `a` à `e`.
- `afficherChiffres()` affiche `1` à `5`.

Chaque affichage est suivi d'une pause de 50 ms. `afficherLettres` est lancée avec `go`, ce qui crée une goroutine. Pendant qu'elle s'exécute, la goroutine principale lance `afficherChiffres`.

Les deux suites peuvent donc s'entrelacer. L'ordre exact peut changer d'une exécution à l'autre, car il dépend de l'ordonnancement des goroutines par le runtime Go.

### Pourquoi le `time.Sleep` final ?

Une fois `afficherChiffres()` terminée, la fonction `main` arrive à sa fin. La fin de `main` arrête immédiatement le programme, y compris les goroutines encore actives.

Le `time.Sleep` final laisse donc du temps à `afficherLettres()` pour finir. C'est utile pour une démonstration, mais ce n'est **pas** une synchronisation fiable : une machine lente pourrait nécessiter un délai différent. L'exercice 2 montre la solution correcte.

---

## Exercice 2 — Synchronisation avec `sync.WaitGroup`

Fichier : `ex2/main.go`

```powershell
go run ./ex2
```

Un `sync.WaitGroup` permet d'attendre la fin d'un ensemble de goroutines sans estimer leur durée avec `time.Sleep`.

Le déroulement est le suivant :

1. `wg.Add(2)` indique que deux tâches doivent être attendues.
2. Les deux fonctions sont lancées dans des goroutines.
3. Chaque fonction exécute `defer wg.Done()`. Lorsque la fonction se termine, le compteur du `WaitGroup` diminue de un.
4. `wg.Wait()` bloque `main` tant que le compteur n'est pas revenu à zéro.

`defer wg.Done()` est préférable à un appel placé en fin de fonction : il sera aussi exécuté si la fonction quitte plus tôt.

---

## Exercice 3 — Somme parallèle avec un channel

Fichier : `ex3/main.go`

```powershell
go run ./ex3
```

Le programme demande la valeur de `n`, crée les nombres de 1 à `n`, puis découpe ce slice en quatre morceaux. Une goroutine calcule la somme de chaque morceau.

Exemple :

```text
Entrez n (entier positif) : 1000
```

Chaque goroutine envoie sa somme partielle dans le channel `resultat` :

```go
resultat <- total
```

La goroutine principale reçoit exactement quatre valeurs :

```go
for i := 0; i < 4; i++ {
    total += <-resultat
}
```

Le résultat attendu est calculé avec la formule de la somme des `n` premiers entiers :

```text
n × (n + 1) / 2
1000 × 1001 / 2 = 500500
```

Résultat attendu :

```text
Somme : 500500 (attendue : 500500)
```

Le channel sert ici à transmettre les résultats entre goroutines sans partager directement la variable `total` entre elles.

---

## Exercice 4 — Worker pool

Fichier : `ex4/main.go`

```powershell
go run ./ex4
```

Un worker pool est un groupe fixe de goroutines qui consomment des tâches depuis une file commune. Ici :

- le channel `jobs` reçoit les entiers de 1 au nombre de jobs saisi ;
- le nombre de workers est saisi au lancement ;
- chaque worker calcule le carré du job ;
- le channel `resultats` transmet les carrés à `main`.

Le flux du programme est :

```text
main → jobs → 4 workers → resultats → main
```

Après l'envoi de tous les jobs, `jobs` est fermé. Les workers terminent alors leur boucle `for job := range jobs` lorsqu'il n'y a plus de valeur à lire.

Le `WaitGroup` attend la fin des quatre workers. Une goroutine ferme ensuite `resultats`, ce qui permet à la boucle de réception suivante de se terminer proprement :

```go
for resultat := range resultats {
    fmt.Println("résultat :", resultat)
}
```

### Pourquoi l'ordre des résultats n'est-il pas garanti ?

Les quatre workers s'exécutent en concurrence. Un job envoyé après un autre peut être traité plus rapidement par un worker différent, puis être envoyé dans `resultats` avant le premier. L'ordonnancement dépend du runtime Go et de l'état de la machine au moment de l'exécution.

Le programme garantit que chaque job traité produit un résultat, mais ne garantit pas l'ordre dans lequel ces résultats arrivent.

---

## Exercice 5 — `select` et timeout

Fichier : `ex5/main.go`

```powershell
go run ./ex5
```

Cet exercice reprend le worker pool. Le worker numéro 4 attend deux secondes avant d'envoyer chacun de ses résultats :

```go
time.Sleep(2 * time.Second)
```

Dans `main`, un `select` attend soit un résultat, soit l'expiration d'un délai de 500 ms :

```go
select {
case resultat, ok := <-resultats:
    // réception normale d'un résultat
case <-time.After(500 * time.Millisecond):
    fmt.Println("timeout sur un résultat")
    return
}
```

`select` choisit le premier cas prêt. Si aucun résultat n'arrive dans les 500 ms, le channel produit par `time.After` reçoit une valeur et le programme affiche :

```text
timeout sur un résultat
```

Selon l'ordonnancement, quelques résultats rapides peuvent être affichés avant le timeout. Dans cette version pédagogique, `main` s'arrête au premier timeout. Le bonus met en place une annulation complète et propre des goroutines restantes.

---

## Exercice 6 — Race condition et `sync.Mutex`

Fichier : `ex6/main.go`

Exécuter la version corrigée :

```powershell
go run ./ex6
```

Le programme lance 1000 goroutines qui incrémentent la même variable `compteur`.

### Le problème sans protection

L'instruction :

```go
compteur++
```

n'est pas atomique. Elle correspond conceptuellement à trois opérations : lire `compteur`, ajouter 1, puis écrire la nouvelle valeur. Deux goroutines peuvent lire la même valeur avant que l'une d'elles ne l'écrive. Une incrémentation est alors perdue.

Sans correction, le résultat est donc variable et peut être inférieur à 1000.

Pour exécuter le détecteur de race :

```powershell
go run -race ./ex6
```

Sur une version non corrigée, Go signale généralement `WARNING: DATA RACE`, avec les accès concurrents à la variable concernée et les goroutines impliquées.

### La correction avec un mutex

Le `sync.Mutex` garantit qu'une seule goroutine à la fois entre dans la section critique :

```go
mutex.Lock()
compteur++
mutex.Unlock()
```

Après la correction, le résultat est stable :

```text
Compteur final : 1000
```

> Dans certains environnements Windows, `go run -race` nécessite CGO activé et un compilateur C disponible. Si Go affiche `-race requires cgo`, il faut installer/configurer un compilateur C puis relancer avec `CGO_ENABLED=1`.

---

## Bonus — Annulation avec `context.Context`

Fichier : `bonus/main.go`

```powershell
go run ./bonus
```

Le programme crée un contexte avec une durée maximale d'une seconde :

```go
ctx, annuler := context.WithTimeout(context.Background(), time.Second)
defer annuler()
```

Le contexte est transmis à tous les workers. Chaque worker surveille `ctx.Done()` : lorsque le délai est atteint, il abandonne son attente éventuelle, ne prend plus de nouveau job et se termine.

Le producteur de jobs surveille aussi le contexte ; il arrête donc l'envoi de jobs si l'annulation survient. Cette approche évite de laisser des goroutines bloquées après que `main` a décidé d'arrêter le traitement.

La différence essentielle avec l'exercice 5 est la suivante :

| Exercice 5 | Bonus |
| --- | --- |
| `main` abandonne après un timeout de réception. | Toutes les goroutines reçoivent un signal d'annulation. |
| Des goroutines peuvent encore travailler jusqu'à la fin du processus. | Les workers restants s'arrêtent proprement. |
| Timeout local à l'attente d'un résultat. | Durée de vie globale du traitement limitée à une seconde. |

## Récapitulatif

| Notion | Exercice | Rôle |
| --- | --- | --- |
| Goroutine | 1 | Exécuter une fonction en concurrence avec `go`. |
| `WaitGroup` | 2 et 4 | Attendre la fin d'un groupe de goroutines. |
| Channel | 3 à 5 | Échanger des données ou synchroniser des goroutines. |
| Worker pool | 4 et 5 | Limiter le nombre de traitements simultanés. |
| `select` et timeout | 5 | Attendre plusieurs événements possibles. |
| `Mutex` | 6 | Protéger une donnée partagée. |
| `context.Context` | Bonus | Propager une annulation ou une échéance. |
