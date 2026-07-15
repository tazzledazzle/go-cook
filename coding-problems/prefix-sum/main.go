package main

// highest altitude
func largestAltitude(gain []int) int {
	current, best := 0, 0
	for _, g := range gain {
		current += g
		if current > best {
			best = current
		}
	}
	return best
}
