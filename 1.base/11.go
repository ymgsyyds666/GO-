package main

import (
	"fmt"
)



func main() {
	s := make([]int,3,5)
	s = append(s,1)
	s = append(s,2,3,4)
	fmt.Println(s)
}