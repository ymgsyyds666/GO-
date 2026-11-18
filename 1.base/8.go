package main
import "fmt"//字符串
func main() {
  s :="string1";
  fmt.Print(len(s));
  fmt.Print(string(s[0]));
  b := []byte(s)
  c := []rune(s)
  fmt.Print(b[0],c[0])
}