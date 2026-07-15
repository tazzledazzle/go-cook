package main

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
)

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

// determine if two strings are close
func closeStrings(word1 string, word2 string) bool {
	if len(word1) != len(word2) {
		return false
	}
	freq1, freq2 := [26]int{}, [26]int{}
	for _, ch := range word1 {
		freq1[ch-'a']++
	}
	for _, ch := range word2 {
		freq2[ch-'a']++
	}
	for i := 0; i < 26; i++ {
		if (freq1[i] == 0) != (freq2[i] == 0) {
			return false
		}
	}
	ch1 := append([]int{}, freq1[:]...)
	ch2 := append([]int{}, freq2[:]...)
	sort.Ints(ch1)
	sort.Ints(ch2)
	return reflect.DeepEqual(ch1, ch2)
}

// equal row and column pairs
func equalPairs(grid [][]int) int {
	n := len(grid)
	rowCount := map[string]int{}
	for _, row := range grid {
		rowCount[encode(row)]++
	}
	total := 0
	for c := 0; c < n; c++ {
		col := make([]int, n)
		for r := 0; r < n; r++ {
			col[r] = grid[r][c]
		}
		total += rowCount[encode(col)]
	}
	return total
}

func encode(row []int) string {
	parts := make([]string, len(row))
	for i, val := range row {
		parts[i] = strconv.Itoa(val)
	}
	return strings.Join(parts, ",")
}
