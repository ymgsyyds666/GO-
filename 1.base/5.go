package main
import "fmt"//自增写法位运算
func main() {
  type w uint8
  const(
    a w=1<<iota
    b
    c
    d
  )
  fmt.Println(a,b,c,d)
}