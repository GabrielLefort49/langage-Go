package main

import (
	"fmt"
	"time"
)

func afficherLettres() {
	for lettre := 'a'; lettre <= 'e'; lettre++ {
		fmt.Printf("%c ", lettre)
		time.Sleep(50 * time.Millisecond)
	}
}

func afficherChiffres() {
	for chiffre := 1; chiffre <= 5; chiffre++ {
		fmt.Printf("%d ", chiffre)
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	go afficherLettres()
	afficherChiffres()

	// Sans ce délai, main peut se terminer avant afficherLettres :
	// une partie (voire aucune) des lettres ne sera alors affichée.
	time.Sleep(300 * time.Millisecond)
	fmt.Println()
}
