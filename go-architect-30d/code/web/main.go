// Package main 演示 03-服务端开发.md 中的 net/http 中间件链与优雅关闭。
// 启动一个临时端口的服务，自我请求验证后优雅退出。
// 运行：go run ./web
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type ctxKey struct{}

func check(cond bool, msg string) {
	if !cond {
		panic("断言失败: " + msg)
	}
}

// withTrace 注入 trace_id：优先复用上游传入的，否则生成。
func withTrace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Trace-ID")
		if id == "" {
			id = fmt.Sprintf("trace-%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Trace-ID", id)
		ctx := context.WithValue(r.Context(), ctxKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// logging 记录访问日志。
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("access",
			"method", r.Method,
			"path", r.URL.Path,
			"cost", time.Since(start),
			"trace_id", r.Context().Value(ctxKey{}),
		)
	})
}

// recoverMW 兜底 panic，避免单个请求打挂进程。
func recoverMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "err", rec, "path", r.URL.Path)
				http.Error(w, `{"code":50000,"message":"internal error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Go 1.22+ 的路径参数语法
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":       r.PathValue("id"),
			"trace_id": r.Context().Value(ctxKey{}).(string),
		})
	})

	mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	return mux
}

func main() {
	// 顺序很重要：TraceID 必须最外层，否则日志里取不到 ctx 中的 trace_id
	handler := withTrace(recoverMW(logging(newMux())))

	// 用 0 端口拿随机可用端口，避免测试时冲突
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	addr := lis.Addr().String()

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		if err := srv.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	fmt.Println("listening on", addr)

	client := &http.Client{Timeout: 2 * time.Second}

	// 自测 1：正常请求，验证 trace_id 透传
	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+"/users/42", nil)
	req.Header.Set("X-Trace-ID", "trace-abc")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		panic(err)
	}
	check(resp.StatusCode == 200, "GET /users/42 应返回 200")
	check(body["id"] == "42", "路径参数解析")
	check(body["trace_id"] == "trace-abc", "trace_id 透传")
	fmt.Println("GET /users/42 ->", body)

	// 自测 2：panic 被中间件兜住
	resp2, err := client.Get("http://" + addr + "/panic")
	if err != nil {
		panic(err)
	}
	defer resp2.Body.Close()
	check(resp2.StatusCode == 500, "panic 应返回 500")
	fmt.Println("GET /panic ->", resp2.StatusCode, "(进程存活)")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
	fmt.Println("graceful shutdown done")
}
