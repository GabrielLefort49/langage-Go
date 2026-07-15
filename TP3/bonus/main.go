package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func worker(ctx context.Context, id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			if id == 4 {
				select {
				case <-time.After(2 * time.Second):
				case <-ctx.Done():
					return
				}
			}
			select {
			case resultats <- job * job:
			case <-ctx.Done():
				return
			}
		}
	}
}

func main() {
	ctx, annuler := context.WithTimeout(context.Background(), time.Second)
	defer annuler()

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
	for id := 1; id <= workers; id++ {
		wg.Add(1)
		go worker(ctx, id, jobs, resultats, &wg)
	}
	go func() {
		defer close(jobs)
		for job := 1; job <= nombreJobs; job++ {
			select {
			case jobs <- job:
			case <-ctx.Done():
				return
			}
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
		case <-ctx.Done():
			fmt.Println("annulation :", ctx.Err())
			return
		}
	}
}
