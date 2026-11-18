package main

import (
	"fmt"
)



func main() {
	var a [3]int
	fmt.Println(a, len(a), cap(a))
	b :=[3]int{1,2,3};
	c :=[...]int{1,2,3,4};
	d :=a
    fmt.Println(b, c,d)
}