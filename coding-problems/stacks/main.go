package main

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
