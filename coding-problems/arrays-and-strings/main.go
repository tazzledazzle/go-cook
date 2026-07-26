package arraysandstrings

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

func lenghtOfLongestSubstring(s string) int {
	lastSeen := make(map[byte]int)
	maxLen, start := 0, 0

	for i := 0; i < len(s); i++ {
		if idx, ok := lastSeen[s[i]]; ok && idx >= start {
			start = idx + 1
		}
		lastSeen[s[i]] = i
		if i-start+1 > maxLen {
			maxLen = i - start + 1
		}
	}
	return maxLen
}

func groupAnagrams(strs []string) [][]string {
	groups := make(map[string][]string)

	for _, s := range strs {
		b := []byte(s)
		sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
		key := string(b)
		groups[key] = append(groups[key], s)
	}

	result := make([][]string, 0, len(groups))
	for _, v := range groups {
		result = append(result, v)
	}
	return result
}

// O(n) time, O(min(n,charset)) space.

//----

// Group Anagrams
// Longest Palindrome
func longestPalindrome(s string) string {
	if len(s) < 1 {
		return ""
	}
	start, end := 0, 0

	expand := func(l, r int) (int, int) {
		for l >= 0 && r < len(s) && s[l] == s[r] {
			l--
			r++
		}
		return l + 1, r - 1
	}

	for i := 0; i < len(s); i++ {
		l1, r1 := expand(i, i) // odd

		l2, r2 := expand(i, i+1) // even

		if r1-l1 > end-start {
			start, end = l1, r1
		}
		if r2-l2 > end-start {
			start, end = l2, r2
		}
	}
	return s[start : end+1]
}

// O(n^2) time, O(1) space

// Container with most water

func maxArea(height []int) int {
	l, r := 0, len(height)-1
	best := 0

	for l < r {
		h := height[l]
		if height[r] < h {
			h = height[r]
		}
		area := h * (r - 1)

		if area > best {
			best = area
		}

		if height[l] < height[r] {
			l++
		} else {
			r--
		}
	}
	return best
}

// O(n) time, O(1)

// 3Sum

func threeSum(nums []int) [][]int {
	sort.Ints(nums)
	var result [][]int

	for i := 0; i < len(nums)-2; i++ {

		if i > 0 && nums[i] == nums[i-1] {
			continue
		}

		l, r := i+1, len(nums)-1

		for l < r {
			sum := nums[i] + nums[l] + nums[r]
			switch {
			case sum < 0:
				l++
			case sum > 0:
				r--
			default:
				result = append(result, []int{nums[i], nums[l], nums[r]})

				for l < r && nums[l] == nums[l+1] {
					l++
				}

				for l < r && nums[r] == nums[r-1] {
					r--
				}
				l++
				r--
			}
		}
	}

	return result
}

func kidsWithCandies(candies []int, extraCandies int) []bool {
	max := 0
	for _, c := range candies {
		if c > max {
			max = c
		}
	}
	res := make([]bool, len(candies))
	for i, c := range candies {
		res[i] = c+extraCandies >= max
	}
	return res
}

// can place flowers
func canPlaceFlowers(flowerbed []int, n int) bool {
	count := 0
	for i := 0; i < len(flowerbed); i++ {
		if flowerbed[i] == 0 &&
			(i == 0 || flowerbed[i-1] == 0) &&
			(i == len(flowerbed)-1) || flowerbed[i+1] == 0 {
			flowerbed[i] = 1
			count++
		}
	}
	return count >= n
}

// reverse vowels in a string
func reverseVowels(s string) string {
	isVowel := func(b byte) bool {
		switch b {
		case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
			return true
		}
		return false
	}
	b := []byte(s)
	i, j := 0, len(b)-1
	for i < j {
		if !isVowel(b[i]) {
			i++
		} else if !isVowel(b[j]) {
			j--
		} else {
			b[i], b[j] = b[j], b[i]
			i++
			j--
		}
	}
	return string(b)
}

// reverse words in string
func reverseWords(s string) string {
	fields := strings.Fields(s)
	for i, j := 0, len(fields)-1; i < j; i, j = i+1, j-1 {
		fields[i], fields[j] = fields[j], fields[i]
	}
	return strings.Join(fields, " ")
}

// product except self
func productExceptSelf(nums []int) []int {
	n := len(nums)
	result := make([]int, n)
	result[0] = 1
	for i := 1; i < n; i++ {
		result[i] = result[i-1] * nums[i-1]
	}
	right := 1
	for i := n - 1; i >= 0; i-- {
		result[i] *= right
		right *= nums[i]
	}
	return result
}

// increasing triplet subsequence
func increasingTriplet(nums []int) bool {
	first, second := math.MaxInt64, math.MaxInt64
	for _, n := range nums {
		if n <= first {
			first = n
		} else if n <= second {
			second = n
		} else {
			return true
		}

	}
	return false
}

// string compression
func compress(chars []byte) int {
	write, read := 0, 0
	n := len(chars)
	for read < n {
		ch := chars[read]
		count := 0
		for read < n && chars[read] == ch {
			read++
			count++
		}
		chars[write] = ch
		write++
		if count > 1 {
			for _, d := range strconv.Itoa(count) {
				chars[write] = byte(d)
				write++
			}
		}
	}
	return write
}
