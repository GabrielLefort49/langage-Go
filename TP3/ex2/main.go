package main

import (
	"fmt"
	"sync"
	"time"
)

func afficherLettres(wg *sync.WaitGroup) {
	defer wg.Done()
	for lettre := 'a'; lettre <= 'e'; lettre++ {
		fmt.Printf("%c ", lettre)
		time.Sleep(50 * time.Millisecond)
	}
}

func afficherChiffres(wg *sync.WaitGroup) {
	defer wg.Done()
	for chiffre := 1; chiffre <= 5; chiffre++ {
		fmt.Printf("%d ", chiffre)
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go afficherLettres(&wg)
	go afficherChiffres(&wg)
	wg.Wait()
	fmt.Println()
}
