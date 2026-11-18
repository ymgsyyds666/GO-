// Package main 演示 01-语言基础.md 中的核心语法点。
// 运行：go run ./basics
package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ---------- 泛型 ----------

type Number interface{ ~int | ~int64 | ~float64 }

func Sum[T Number](xs []T) T {
	var total T
	for _, x := range xs {
		total += x
	}
	return total
}

func Map[T, U any](xs []T, f func(T) U) []U {
	out := make([]U, 0, len(xs))
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}

// ---------- 错误处理 ----------

var ErrNotFound = errors.New("not found")

type ValidationError struct{ Field string }

func (e *ValidationError) Error() string { return "invalid field: " + e.Field }

func findUser(id int) error {
	return fmt.Errorf("find user %d: %w", id, ErrNotFound)
}

// ---------- 接口与方法集 ----------

type Speaker interface{ Speak() string }

type Dog struct{ Name string }

func (d *Dog) Speak() string { return d.Name + " says woof" }
func (d Dog) String() string { return "Dog(" + d.Name + ")" }

func check(cond bool, msg string) {
	if !cond {
		panic("断言失败: " + msg)
	}
}

func section(name string) { fmt.Printf("\n== %s ==\n", name) }

func main() {
	section("泛型")
	check(Sum([]int{1, 2, 3}) == 6, "Sum int")
	check(Sum([]float64{1.5, 2.5}) == 4.0, "Sum float")
	fmt.Println("Sum:", Sum([]int{1, 2, 3}), Sum([]float64{1.5, 2.5}))
	fmt.Println("Map:", Map([]int{1, 2, 3}, func(x int) string {
		return strings.Repeat("x", x)
	}))

	section("切片扩容")
	s := make([]int, 0)
	for i := 0; i < 5; i++ {
		s = append(s, i)
		fmt.Printf("len=%d cap=%d\n", len(s), cap(s))
	}
	check(len(s) == 5 && cap(s) >= 5, "slice 长度")

	section("切片共享底层数组")
	arr := [5]int{1, 2, 3, 4, 5}
	s1 := arr[1:3]
	s1[0] = 99
	check(arr[1] == 99, "切片共享底层数组")
	fmt.Println("arr:", arr, "s1 len/cap:", len(s1), cap(s1))

	section("map")
	m := map[string]int{"a": 1}
	v, ok := m["a"]
	check(ok && v == 1, "map 命中")
	_, ok = m["nope"]
	check(!ok, "map 未命中")
	delete(m, "a")
	check(len(m) == 0, "map 删除")

	set := map[string]struct{}{}
	set["go"] = struct{}{}
	_, ok = set["go"]
	check(ok, "map 当集合用")

	section("字符串")
	text := "Go语言"
	check(len(text) == 8, "字节长度")
	count := 0
	for range text {
		count++
	}
	check(count == 4, "rune 数量")
	fmt.Println("字节 len:", len(text), "字符数:", count)

	var sb strings.Builder
	for i := 0; i < 3; i++ {
		sb.WriteString("go")
	}
	check(sb.String() == "gogogo", "Builder")
	fmt.Println("Builder:", sb.String())

	section("错误链")
	err := findUser(7)
	check(errors.Is(err, ErrNotFound), "errors.Is 沿链查找")
	var ve *ValidationError
	wrapped := fmt.Errorf("wrap: %w", &ValidationError{Field: "email"})
	check(errors.As(wrapped, &ve) && ve.Field == "email", "errors.As 提取类型")
	fmt.Println("err:", err)
	fmt.Println("As:", ve.Field)

	section("接口 nil 陷阱")
	var typedNil *ValidationError
	var e error = typedNil
	check(e != nil, "typed nil 装进接口后不等于 nil")
	fmt.Println("接口 == nil ?", e == nil)
	// 正确做法：失败时显式返回 nil
	check(good() == nil, "显式返回 nil")

	section("方法集")
	var sp Speaker = &Dog{Name: "wangcai"}
	fmt.Println(sp.Speak(), "|", Dog{Name: "wangcai"})
	check(sp.Speak() == "wangcai says woof", "接口实现")

	section("defer 顺序")
	var order []int
	func() {
		for i := 0; i < 3; i++ {
			defer func(n int) { order = append(order, n) }(i)
		}
	}()
	check(fmt.Sprint(order) == "[2 1 0]", "defer LIFO")
	fmt.Println("defer 执行顺序:", order)

	section("排序")
	nums := []int{3, 1, 2}
	sort.Ints(nums)
	check(fmt.Sprint(nums) == "[1 2 3]", "sort.Ints")
	fmt.Println(nums)
}

// 正确示范：没有错误就返回 untyped nil，接口值才真的等于 nil。
func good() error { return nil }
