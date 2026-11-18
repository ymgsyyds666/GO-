// Package main 演示 02-并发与工程化.md 中的并发模式。
// 运行：go run ./concurrency
package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func check(cond bool, msg string) {
	if !cond {
		panic("断言失败: " + msg)
	}
}

func section(name string) { fmt.Printf("\n== %s ==\n", name) }

// ---------- Worker Pool ----------

func workerPool(jobs []int, workers int) []int {
	jobCh := make(chan int, len(jobs))
	resCh := make(chan int, len(jobs))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				resCh <- j * 2
			}
		}()
	}

	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	go func() {
		wg.Wait()
		close(resCh)
	}()

	var out []int
	for r := range resCh {
		out = append(out, r)
	}
	sort.Ints(out)
	return out
}

// ---------- Pipeline ----------

func gen(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			select {
			case out <- n * n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func sum(ctx context.Context, in <-chan int) int {
	total := 0
	for {
		select {
		case n, ok := <-in:
			if !ok {
				return total
			}
			total += n
		case <-ctx.Done():
			return total
		}
	}
}

// ---------- 令牌桶限流 ----------

type TokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   time.Now(),
	}
}

func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.tokens = min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// ---------- 超时控制 ----------

func slowWork(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	section("Worker Pool")
	got := workerPool([]int{1, 2, 3, 4, 5}, 3)
	check(fmt.Sprint(got) == "[2 4 6 8 10]", "worker pool 结果")
	fmt.Println("输入 1..5，3 个 worker，输出:", got)

	section("Pipeline")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	total := sum(ctx, square(ctx, gen(ctx, 1, 2, 3, 4, 5)))
	check(total == 55, "1^2+...+5^2 = 55")
	fmt.Println("1^2+2^2+...+5^2 =", total)

	section("令牌桶")
	tb := NewTokenBucket(100, 3)
	allowed := 0
	for i := 0; i < 10; i++ {
		if tb.Allow() {
			allowed++
		}
	}
	check(allowed == 3, "burst=3 时前 10 次只放行 3 次")
	fmt.Println("burst=3，连续 10 次请求放行:", allowed)

	time.Sleep(50 * time.Millisecond) // 100/s * 50ms = 5 个令牌，被 burst 截断为 3
	allowed = 0
	for i := 0; i < 3; i++ {
		if tb.Allow() {
			allowed++
		}
	}
	check(allowed == 3, "补充令牌后放行 3 次")
	fmt.Println("等待 50ms 后再请求 3 次，放行:", allowed)

	section("context 超时")
	tctx, tcancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer tcancel()
	err := slowWork(tctx, 100*time.Millisecond)
	check(errors.Is(err, context.DeadlineExceeded), "超时应返回 DeadlineExceeded")
	fmt.Println("10ms 超时，100ms 任务返回:", err)

	section("atomic 计数")
	var counter atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()
	check(counter.Load() == 1000, "atomic 计数正确")
	fmt.Println("1000 个协程计数:", counter.Load())

	section("WaitGroup.Go (Go 1.25)")
	var wg2 sync.WaitGroup
	var sum2 atomic.Int64
	for i := 1; i <= 100; i++ {
		wg2.Go(func() { sum2.Add(int64(i)) })
	}
	wg2.Wait()
	check(sum2.Load() == 5050, "WaitGroup.Go 求和")
	fmt.Println("1..100 求和:", sum2.Load())

	section("goroutine 数量")
	fmt.Println("当前协程数:", runtime.NumGoroutine())
}
