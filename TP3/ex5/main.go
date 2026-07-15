package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		if id == 4 { // un worker sur quatre est volontairement lent
			time.Sleep(2 * time.Second)
		}
		resultats <- job * job
	}
}

func main() {
	var workers, nombreJobs int
	fmt.Print("Nombre de workers (au moins 4) : ")
	if _, err := fmt.Scan(&workers); err != nil || workers < 4 {
		fmt.Println("Le nombre de workers doit être supérieur ou égal à 4.")
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
		defer close(jobs)
		for job := 1; job <= nombreJobs; job++ {
			jobs <- job
		}
	}()
	go func() {
		wg.Wait()
		close(resultats)
	}()

	for {
		select {
		case resultat, ok := <-resultats:
			if !ok {
				return
			}
			fmt.Println("résultat :", resultat)
		case <-time.After(500 * time.Millisecond):
			fmt.Println("timeout sur un résultat")
			return
		}
	}
}
