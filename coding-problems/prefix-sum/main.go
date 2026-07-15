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

// find pivot index
func pivotIndex(nums []int) int {
	total := 0
	for _, num := range nums {
		total += num
	}
	leftSum := 0
	for i, num := range nums {
		if leftSum == total-leftSum-num {
			return i
		}
		leftSum += num
	}
	return -1
}
