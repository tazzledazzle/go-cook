package main

// difference between two arrays
func findDifference(nums1 []int, nums2 []int) [][]int {
	s1, s2 := map[int]bool{}, map[int]bool{}

	for _, n := range nums1 {
		s1[n] = true
	}
	for _, n := range nums2 {
		s2[n] = true
	}
	result := [][]int{{}, {}}
	for num := range s1 {
		if !s2[num] {
			result[0] = append(result[0], num)
		}
	}
	for num := range s2 {
		if !s1[num] {
			result[1] = append(result[1], num)
		}
	}
	return result
}

// unique number of occurrences
func uniqueOccurrences(arr []int) bool {
	freq := map[int]int{}
	for _, num := range arr {
		freq[num]++
	}
	seen := map[int]bool{}
	for _, num := range freq {
		if seen[num] {
			return false
		}
		seen[num] = true
	}
	return true
}
