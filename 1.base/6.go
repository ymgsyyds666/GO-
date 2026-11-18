package main
import "fmt"//自增写法位运算
func main() {
  var i int = 10
  var f float64 = float64(i)  
  //var f float64 = i  
   // 必须显式,cpp
  var u uint = uint(i)
  fmt.Println(i,f,u)
}