package main

import "fmt"

func somme(nombres []int, resultat chan<- int) {
	total := 0
	for _, nombre := range nombres {
		total += nombre
	}
	resultat <- total
}

func main() {
	const morceaux = 4
	var n int

	fmt.Print("Entrez n (entier positif) : ")
	if _, err := fmt.Scan(&n); err != nil || n <= 0 {
		fmt.Println("Veuillez saisir un entier strictement positif.")
		return
	}

	nombres := make([]int, n)
	for i := range nombres {
		nombres[i] = i + 1
	}

	resultat := make(chan int)
	// On accepte aussi les valeurs de n qui ne sont pas divisibles par 4.
	tailleMorceau := (n + morceaux - 1) / morceaux
	for i := 0; i < morceaux; i++ {
		debut := i * tailleMorceau
		if debut >= n {
			go somme(nil, resultat)
			continue
		}
		fin := min(debut+tailleMorceau, n)
		go somme(nombres[debut:fin], resultat)
	}

	total := 0
	for i := 0; i < morceaux; i++ {
		total += <-resultat
	}

	attendu := n * (n + 1) / 2
	fmt.Printf("Somme : %d (attendue : %d)\n", total, attendu)
}
