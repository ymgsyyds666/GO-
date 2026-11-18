// Package main 演示 04-AI与OCR.md 中的 LLM SSE 流式接入。
// 用 httptest 起一个假的 OpenAI 兼容服务，验证流式解析逻辑，无需真实 API Key。
// 运行：go run ./llm
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

func check(cond bool, msg string) {
	if !cond {
		panic("断言失败: " + msg)
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

type chunkResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// StreamChat 调用 OpenAI 兼容的 /v1/chat/completions 流式接口。
// 每收到一个增量片段就回调 onDelta，适合直接转发给前端 SSE。
func StreamChat(ctx context.Context, baseURL, apiKey, model, prompt string,
	onDelta func(string)) error {

	body, err := json.Marshal(ChatRequest{
		Model:    model,
		Messages: []Message{{Role: "user", Content: prompt}},
		Stream:   true,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "text/event-stream")

	// 流式请求不设整体超时，用 ctx 控制生命周期
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("llm status=%d body=%s", resp.StatusCode, b)
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return nil
		}

		var chunk chunkResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // 跳过心跳/注释行
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onDelta(chunk.Choices[0].Delta.Content)
		}
	}
	return sc.Err()
}

// fakeLLM 返回一个按 SSE 逐字吐出的假服务。
func fakeLLM(parts []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		for _, p := range parts {
			payload, _ := json.Marshal(map[string]any{
				"choices": []any{
					map[string]any{"delta": map[string]string{"content": p}},
				},
			})
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush() // 关键：每片都立即下发
			time.Sleep(5 * time.Millisecond)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

func main() {
	srv := fakeLLM([]string{"你好", "，", "我是", " Go ", "助手", "。"})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var got strings.Builder
	start := time.Now()
	err := StreamChat(ctx, srv.URL, "test-key", "fake-model", "打个招呼",
		func(delta string) {
			got.WriteString(delta)
			fmt.Printf("[+%4dms] %s\n", time.Since(start).Milliseconds(), delta)
		})
	if err != nil {
		panic(err)
	}

	check(got.String() == "你好，我是 Go 助手。", "流式内容拼接正确")
	fmt.Println("\n完整回答:", got.String())
	fmt.Printf("总耗时: %v（6 个分片，服务端每片间隔 5ms）\n", time.Since(start))

	// 错误路径：错误的 key 应返回明确错误
	errSrv := fakeLLM([]string{"x"})
	defer errSrv.Close()

	err = StreamChat(ctx, errSrv.URL, "wrong-key", "m", "hi", func(string) {})
	check(err != nil && strings.Contains(err.Error(), "401"), "401 应返回错误")
	fmt.Println("\n鉴权失败返回:", err)
}
