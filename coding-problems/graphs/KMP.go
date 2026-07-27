package main

import "fmt"

func KMP_PrefixTable(P string) (F []int) {
	F = make([]int, len(P))
	pos, comp := 2, 0
	F[0], F[1] = -1, 0
	for pos < len(P) {
		if P[pos-1] == P[comp] {
			comp++
			F[pos] = comp
			pos++
		} else if comp > 0 {
			comp = F[comp]
		} else {
			F[pos] = 0
			pos++
		}
	}
	return F
}

func KMP(T, P string) {
	m, i, c := 0, 0, 0
	F := KMP_PrefixTable(P)
	for m+i < len(T) {
		fmt.Printf("\ncomparing characters %c $%c at positions %d %d", T[m+i], P[i], m+i, i)
		c++
		if P[i] == T[m+i] {
			fmt.Printf(" - match")
			if i == len(P)-1 {
				fmt.Printf("\n\nWord %q was found at position %d in %q with %d comparisons.", P, m, T, c)
				return
			}
			i++
		} else {
			m = m + i - F[i]
			if F[i] > -1 {
				i = F[i]
			} else {
				i = 0
			}
		}
	}
	fmt.Printf("\n\nWord was not found. \n%d comparisons were done.", c)
}
