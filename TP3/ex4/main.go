package main

import (
	"fmt"
	"sync"
)

func worker(id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		fmt.Printf("worker %d traite %d\n", id, job)
		resultats <- job * job
	}
}

func main() {
	var workers, nombreJobs int
	fmt.Print("Nombre de workers : ")
	if _, err := fmt.Scan(&workers); err != nil || workers <= 0 {
		fmt.Println("Le nombre de workers doit être strictement positif.")
		return
	}
	fmt.Print("Nombre de jobs : ")
	if _, err := fmt.Scan(&nombreJobs); err != nil || nombreJobs <= 0 {
		fmt.Println("Le nombre de jobs doit être strictement positif.")
		return
	}

	jobs := make(chan int)
	resultats := make(chan int)
	var wg sync.WaitGroup

	wg.Add(workers)
	for id := 1; id <= workers; id++ {
		go worker(id, jobs, resultats, &wg)
	}

	go func() {
		for job := 1; job <= nombreJobs; job++ {
			jobs <- job
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(resultats)
	}()

	for resultat := range resultats {
		// L'ordonnancement des goroutines varie : le worker qui finit en premier
		// envoie son résultat en premier, donc l'ordre n'est pas garanti.
		fmt.Println("résultat :", resultat)
	}
}
