package main

import (
	"fmt"
	"sync"
)

func main() {
	compteur := 0
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mutex.Lock()
			compteur++
			mutex.Unlock()
		}()
	}

	wg.Wait()
	fmt.Println("Compteur final :", compteur)
}

// Sans mutex, le résultat peut être inférieur à 1000 : plusieurs goroutines
// lisent puis écrivent compteur simultanément. go run -race signale des
// "DATA RACE". Avec le mutex, le résultat est toujours 1000.
