package main

import (
	"fmt"
	"io/fs"
	"time"
)

func main() {
	now := time.Now()
	fmt.Println(now)

}

func SumAndProduct(A, B int) (int, int) {
	return A + B, A * B
}

// defer

func ReadWrite(file fs.File) bool {
	//file.Open("file")
	//	defer file.Close()
	return false
}

// struct

type person struct {
	name string
	age  int
}
