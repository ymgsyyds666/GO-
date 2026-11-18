package main
import "fmt"
func main() {
var m map[string]int // nil map：可读，写入会 panic！
m["a"] = 1 
fmt.Println(m["a"])//类型错误，强迫类型
  
}