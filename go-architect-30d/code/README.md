# 配套可运行代码

全部基于 **Go 标准库**，不需要下载任何第三方依赖，`go run` 即可。

| 目录 | 演示内容 | 对应文档 |
|---|---|---|
| `basics/` | 泛型、切片扩容与共享、map、字符串/rune、错误链、接口 nil 陷阱、方法集、defer 顺序 | [01-语言基础.md](../01-语言基础.md) |
| `concurrency/` | worker pool、pipeline、令牌桶限流、context 超时、atomic、`WaitGroup.Go` | [02-并发与工程化.md](../02-并发与工程化.md) |
| `web/` | 中间件链（TraceID/日志/Recover）、`ServeMux` 路径参数、优雅关闭 | [03-服务端开发.md](../03-服务端开发.md) |
| `ocr/` | 灰度化 + Otsu 二值化（OCR 预处理核心步骤），输出二值图到 `out/` | [04-AI与OCR.md](../04-AI与OCR.md) |
| `llm/` | OpenAI 兼容协议的 SSE 流式解析（用 httptest 起假服务，无需 Key） | [04-AI与OCR.md](../04-AI与OCR.md) |

## 运行

```bash
cd code
go run ./basics
go run ./concurrency
go run ./web
go run ./ocr
go run ./llm
```

每个程序都内置了断言（`check`），**跑通 = 行为符合文档描述**。

## 常见问题

**1. 报错 `executable file not found` 或 `不是此操作系统平台的有效应用程序`**

说明 `go env GOOS` 被设置成了非当前系统（比如为了交叉编译服务器而设成了 `linux`）。临时覆盖即可：

```powershell
# Windows
$env:GOOS='windows'; $env:GOARCH='386'; go run ./basics

# macOS / Linux
GOOS=$(go env GOHOSTOS) GOARCH=$(go env GOHOSTARCH) go run ./basics
```

**2. 想加自己的练习**

在 `code/` 下新建目录（`package main`），`go run ./你的目录` 即可，不会影响其他示例。

**3. 下一步**

把这些示例改坏再修好：去掉 `defer cancel()`、把 `wg.Add(1)` 挪进 goroutine、把缓冲 channel 改成无缓冲、把 Otsu 换成固定阈值 128 —— 观察现象，这比读十遍文档有用。
