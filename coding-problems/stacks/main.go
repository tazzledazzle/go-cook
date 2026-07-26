package main

import "strings"

// removing stars from a string
func removeStars(s string) string {
	stack := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			stack = stack[:len(stack)-1]
		} else {
			stack = append(stack, s[i])
		}
	}
	return string(stack)
}

// asteroid collision

func asteroidCollision(asteroids []int) []int {
	stack := make([]int, 0, len(asteroids))
	for _, asteroid := range asteroids {
		alive := true
		for alive &&
			asteroid < 0 &&
			len(stack) > 0 &&
			stack[len(stack)-1] > 0 {
			top := stack[len(stack)-1]
			if top < -asteroid {
				stack = stack[:len(stack)-1]
			} else if top == -asteroid {
				stack = stack[:len(stack)-1]
				alive = false
			} else {
				alive = false
			}
		}
		if alive {
			stack = append(stack, asteroid)
		}
	}
	return stack
}

// decode string
func decodeString(s string) string {
	countStack := []int{}
	strStack := []string{}
	current := ""
	num := 0
	for _, ch := range s {
		switch {
		case ch >= '0' && ch <= '9':
			num = num*10 + int(ch-'0')
		case ch == '[':
			countStack = append(countStack, num)
			strStack = append(strStack, current)
			num = 0
			current = ""
		case ch == ']':
			k := countStack[len(countStack)-1]
			countStack = countStack[:len(countStack)-1]
			previous := strStack[len(strStack)-1]
			strStack = strStack[:len(strStack)-1]
			current = previous + strings.Repeat(current, k)
		default:
			current += string(ch)
		}
	}
	return current
}
