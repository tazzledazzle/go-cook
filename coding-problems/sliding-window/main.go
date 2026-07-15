package main

import "strings"

// maximum average subarray
func findMaxAverage(nums []int, k int) float64 {
	sum := 0
	for i := 0; i < k; i++ {
		sum += nums[i]
	}
	best := sum
	for i := k; i < len(nums); i++ {
		sum += nums[i] - nums[i-k]
		if sum > best {
			best = sum
		}
	}
	return float64(best) / float64(k)
}

// max num of vowels ina substring of given length
func maxVowels(s string, k int) int {
	isVowel := func(b byte) bool {
		return strings.ContainsRune("aeiou", rune(b))
	}
	count := 0
	for i := 0; i < k; i++ {
		if isVowel(s[i]) {
			count++
		}
	}
	best := count
	for i := k; i < len(s); i++ {
		if isVowel(s[i]) {
			count++
		}
		if isVowel(s[i-k]) {
			count--
		}
		if count > best {
			best = count
		}
	}

	return best
}
