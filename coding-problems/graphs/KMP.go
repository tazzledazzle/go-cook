package main

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
