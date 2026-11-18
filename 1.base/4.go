package main
import "fmt"//自增写法
func main() {
  type w int
  const(
    a w=iota
    b
    c
  )
  fmt.Println(a,b,c)
}