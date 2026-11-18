package main
import "fmt"
type w int
func main() {
   
  const(
    a w=iota
    b
    c
  )
  fmt.Println(a,b,c)
}