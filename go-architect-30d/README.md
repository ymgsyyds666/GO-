# Go 架构师 30 天冲刺手册

> **目标**：30 天后，你能独立设计、开发、部署一个 Go 服务端项目，并把 AI / OCR 能力接进生产系统，面试时能讲清楚"为什么这么设计"。
>
> **前提**：每天 3~4 小时，其中 **至少 2 小时在写代码**。只读不敲，等于没学。

---

## 0. 三条铁律（先看这个，再看目录）

1. **只跑通不算会，能改才算会。** 每段示例代码，敲完后必须做一次"破坏性修改"：改参数、改并发数、去掉一行看会不会崩。
2. **先跑通，再优雅。** 第一遍全用 `log.Printf` + `println` 也行，能跑起来再谈分层、接口、依赖注入。
3. **每天一个 commit，每周一个可运行的项目。** 你的 GitHub 提交记录，就是简历。

另外两个"不要"：

- 不要一上来学 Kubernetes、Service Mesh、DDD 全套。**先能写出一个正确、不崩、能扛住压测的 CRUD 服务**，再谈架构。
- 不要同时学三个框架。**Gin 打底 → go-zero 或 Kratos 进阶**，其余用到再查。

---

## 0.1 AI 使用公约（重要）

> 这 30 天的目标是"长出能力"，不是"交出项目"。AI 能让你更快交付，也能让你更熟练地"不会"。

**分阶段规则**

| 阶段 | 规则 | 原因 |
|---|---|---|
| 语法练习、LeetCode | **禁用**。卡住先查官方文档/源码 20 分钟 | 语法必须变成肌肉记忆 |
| 并发、内存、调度 | 只用来"解释概念"，不让它写代码 | 这层靠推理，抄一遍等于没学 |
| 服务端业务代码 | 可查 API 用法、生成样板、写测试用例 | 样板不产生能力差异 |
| 核心链路（缓存/发号/事务/限流） | **自己写**，写完再让 AI review | 这是面试和排障的主战场 |
| 架构设计 | 把 AI 当"挑刺的人"，让它反驳你的方案 | 训练取舍，而不是要标准答案 |
| 三个项目 | 设计自己定，非核心可加速；**每一行都要能解释** | 作品集要经得起追问 |

**三条操作纪律**

1. **先写后问**：任何问题先自己尝试 30 分钟，并把"我试了什么、我猜是什么原因"写下来，再问 AI。
2. **问为什么，不要问"给我代码"**：让它解释机制、对比方案、指出你的错误；直接要代码等于放弃思考。
3. **合上重写**：看懂答案后关掉对话，自己重写一遍，再做一次破坏性修改验证理解。

**唯一硬标准**：关掉 AI 和自动补全，40 分钟内手写一个带缓存、带超时、带测试的接口。写不出来，说明那部分还不是你的。

---

## 1. 目录

| 文件 | 内容 | 对应天数 |
|---|---|---|
| [01-语言基础.md](01-语言基础.md) | 语法全貌 + 每节配套练习 + 刷题网站 + 题单映射 | D1–D7 |
| [02-并发与工程化.md](02-并发与工程化.md) | GMP、channel、sync、context、测试、pprof、泛型反射 | D8–D14 |
| [03-服务端开发.md](03-服务端开发.md) | HTTP、Gin、go-zero/Kratos、MySQL/Redis/MQ、gRPC、可观测、部署 | D15–D21 |
| [04-AI与OCR.md](04-AI与OCR.md) | OCR 四条路线、PaddleOCR/Tesseract/ONNX、LLM、RAG、流式输出 | D22–D26 |
| [05-架构师之路.md](05-架构师之路.md) | 容量评估、高并发、稳定性、DDD-lite、选型与 Code Review | D27–D28 |
| [06-实战项目.md](06-实战项目.md) | 三个递进项目：GoScan → 短链服务 → OCR+LLM 文档平台 | 贯穿全程 |
| [index.html](index.html) | 可视化导航页 + 30 天打卡表（可直接用浏览器打开） | — |

---

## 2. 30 天总表

**第一周：语言地基（每天 3h，2h 敲代码 + 1h 刷题）**

| 天 | 主题 | 必敲内容 | 验收标准 |
|---|---|---|---|
| D1 | 环境 + 变量与类型 | 装 Go、配 GOPROXY、写 hello、`go run/build/mod` | 能解释 `go.mod` 每一行 |
| D2 | 流程控制 + 函数 | for/switch/defer/闭包/多返回值 | 能手写 defer 执行顺序 |
| D3 | 数组、切片、map | 切片扩容实验、map 并发 panic 复现 | 能画出 slice 的 ptr/len/cap |
| D4 | 结构体 + 方法 + 接口 | 嵌入、方法集、接口断言、Option 模式 | 能解释"接口值为 nil 但不等 nil" |
| D5 | 错误处理 + 泛型 | `%w`、errors.Is/As、自定义 error、泛型约束 | 能写一个泛型 `Map/Filter` |
| D6 | 标准库实战 | strings/strconv/time/os/io/json/slog/embed | 写一个读 JSON 配置的 CLI |
| D7 | **周项目 1** + 刷题 | GoScan 目录扫描器 v0（单协程）+ LeetCode 20 题 | 项目能跑、题能 AC |

**第二周：并发与工程化（Go 的灵魂）**

| 天 | 主题 | 必敲内容 | 验收标准 |
|---|---|---|---|
| D8 | goroutine + GMP | 10 万协程实验、`GOMAXPROCS`、调度图 | 能讲清 G/M/P 和 work stealing |
| D9 | channel + select | 无缓冲/有缓冲/关闭/单向、超时、fan-in/out | 能写超时控制不泄漏 |
| D10 | sync + atomic | Mutex/RWMutex/WaitGroup/Once/Pool/Map | 用 Pool 优化高频分配 |
| D11 | context | 取消传播、超时、Value 边界、HTTP 集成 | 能写三级超时链 |
| D12 | 并发模式 | worker pool、pipeline、errgroup、singleflight、限流 | 手写令牌桶限流器 |
| D13 | 测试与性能 | 表格测试、fuzz、benchmark、pprof、trace | 用 pprof 找出热点函数 |
| D14 | **周项目 2** | GoScan v1：worker pool + 限流 + 超时 + 优雅退出 + pprof | 压测 1w 文件无泄漏 |

**第三周：服务端（公司的饭碗）**

| 天 | 主题 | 必敲内容 | 验收标准 |
|---|---|---|---|
| D15 | HTTP 原理 + net/http | 手写路由、中间件、优雅关闭 | 不查文档写出 Server 超时配置 |
| D16 | Gin 核心 | 路由/绑定校验/中间件/统一响应/错误码 | 30 分钟搭出带 JWT 的 API |
| D17 | MySQL + GORM/sqlx | 连接池、事务、分页、索引、迁移 | 能写出 3 表关联查询 |
| D18 | Redis + 缓存设计 | go-redis、分布式锁、缓存三兄弟 | 讲清缓存穿透/击穿/雪崩方案 |
| D19 | gRPC + protobuf | proto 编写、server/client、拦截器、超时 | 两个服务能互相调用 |
| D20 | 可观测性 + 部署 | slog/zap、Prometheus、OTel、Docker 多阶段 | 镜像 < 30MB、有 /metrics |
| D21 | **周项目 3** | 短链服务：Gin+MySQL+Redis+压测 | QPS 报告 + 瓶颈分析 |

**第四周：架构与 AI（差异化竞争力）**

| 天 | 主题 | 必敲内容 | 验收标准 |
|---|---|---|---|
| D22 | 架构方法 | 容量估算、分层、DDD-lite、API 设计 | 能画短链系统架构图 |
| D23 | 稳定性 | 限流熔断降级、幂等、SLO、故障演练 | 写出应急预案文档 |
| D24 | OCR 路线选型 | 云 API / Tesseract / PaddleOCR / ONNX | 跑通 2 种 OCR 并对比 |
| D25 | LLM 接入 | OpenAI 兼容 API、SSE 流式、结构化输出 | 流式接口不卡顿、可中断 |
| D26 | RAG + 工程化 | 切分、embedding、向量检索、成本控制 | 文档问答 demo 能答对 |
| D27 | **周项目 4** | OCR+LLM 文档处理平台 | 端到端跑通一条发票 |
| D28 | 架构复盘 | 写 ADR、画时序图、Code Review 清单 | 能评审别人的 PR |
| D29 | 查漏补缺 | 自检清单过一遍、补薄弱项 | 清单 80% 打勾 |
| D30 | 输出 | 写技术博客 + 整理简历项目 + 模拟面试 | 能讲 10 分钟项目 |

> 节奏建议：**上午看书/文档 40 分钟 → 下午敲 2 小时 → 晚上刷题/复盘 40 分钟**。周末各留半天补进度。

---

## 3. 环境准备（D1 完成）

```bash
# 1. 安装 Go（1.22+，建议最新稳定版）后配置国内代理
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.org

# 2. 验证
go version        # go version go1.25.x ...
go env GOPATH

# 3. 编辑器二选一
#   - GoLand（最省心，学生可申请免费）
#   - VS Code + Go 插件（装 gopls、delve）
```

必备工具（陆续装，别一次装完）：

| 工具 | 用途 | 安装 |
|---|---|---|
| `golangci-lint` | 静态检查全家桶 | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| `dlv` | 断点调试 | `go install github.com/go-delve/delve/cmd/dlv@latest` |
| `air` | 热重载 | `go install github.com/air-verse/air@latest` |
| `protoc` + `buf` | gRPC 代码生成 | 官网下载 + `go install github.com/bufbuild/buf/cmd/buf@latest` |
| `hey` / `wrk` | 压测 | 二进制下载 |
| `docker` | 部署 | Docker Desktop |

---

## 4. 每天怎么学（一个可复制的 3 小时模板）

```
00:00–00:40  读：当天文档 + 官方文档对应章节（只读关键 API，不求全）
00:40–02:30  敲：跟着示例写，然后脱离示例重写一遍；再做 1 个破坏性实验
02:30–03:00  刷题 / 写笔记：当天语法点对应的 1~2 道题；用 5 句话总结"今天学会了什么、哪里卡住"
```

**笔记模板**（每天 5 行，30 天后就是你的知识库）：

```markdown
## D08 goroutine
- 会了：go f() 的启动开销约 2KB 栈，可动态扩缩
- 踩坑：循环变量捕获（1.22 前）、goroutine 泄漏
- 代码：./code/concurrency/workerpool.go
- 疑问：GMP 的 handoff 具体触发条件？
- 明日：channel
```

---

## 5. 阶段自检（不到 80% 不许进下一周）

**Week 1 自检**

- [ ] 能解释值类型和引用类型在传参时的差异（用 slice 举例）
- [ ] 能说清 `nil slice` 和 `empty slice` 的区别
- [ ] 能写出接口断言 + `comma-ok`，并解释 nil 接口陷阱
- [ ] 能用 `errors.Is/As/%w` 做错误链
- [ ] 能手写一个泛型函数并解释约束
- [ ] LeetCode 简单题 20 道 AC（Go 语言）

**Week 2 自检**

- [ ] 能画 GMP 调度图，解释为什么 Go 协程比线程轻
- [ ] 能写带超时、可取消、不泄漏的并发任务
- [ ] 能复现 data race 并用 `-race` 定位
- [ ] 能用 pprof 找出 CPU/内存热点
- [ ] 能手写 worker pool 和令牌桶

**Week 3 自检**

- [ ] 能手写 HTTP 中间件链和优雅关闭
- [ ] 能说出 MySQL 连接池三个关键参数
- [ ] 能讲清缓存穿透/击穿/雪崩的差别和方案
- [ ] 能写 proto 并生成 gRPC 代码
- [ ] 能写多阶段 Dockerfile 并用 compose 起全套

**Week 4 自检**

- [ ] 能对一个需求做容量估算（QPS、存储、带宽）
- [ ] 能画出带缓存、队列、降级的系统架构图
- [ ] 能讲清 OCR 三条路线的取舍
- [ ] 能写 LLM 流式接口并对接前端
- [ ] 能独立完成一次 Code Review 并给出改进建议

---

## 6. 常见误区（提前避坑）

| 误区 | 后果 | 正确做法 |
|---|---|---|
| 收藏 20 个教程，一个没敲完 | 30 天后还是小白 | 只跟一套主线，其余按需查 |
| 过早学微服务/DDD | 写不出能跑的服务 | 单体分层 → 压测 → 再拆分 |
| 迷信框架，不懂 net/http | 出问题无法定位 | 先手写一遍路由和中间件 |
| 不看官方文档，只看二手博客 | 学到过期 API | `pkg.go.dev` + 官方 blog 优先 |
| 不做压测就谈高并发 | 上线即崩 | 每个项目必跑 `hey` |
| 代码不格式化、不写测试 | 简历项目没含金量 | `gofmt` + 表格测试是底线 |

---

## 7. 资源速查

**官方**
- [A Tour of Go](https://go.dev/tour/)（交互式，2 小时过一遍）
- [Go by Example](https://gobyexample.com/)（示例驱动，最好的速查）
- [Effective Go](https://go.dev/doc/effective_go)（必读，D3 前读完）
- [Go 官方博客](https://go.dev/blog/)（并发模式、泛型、GC 演进）
- [pkg.go.dev](https://pkg.go.dev/)（标准库唯一真相）

**刷题/练习**
- [Exercism Go Track](https://exercism.org/tracks/go)（130+ 练习，有 mentor，最适合"敲"）
- [LeetCode 中国站](https://leetcode.cn/)（热题 100，按标签刷）
- [Gophercises](https://gophercises.com/)（20 个小项目，免费）
- [Codewars](https://www.codewars.com/)（kata 模式，趣味性强）
- [HackerRank Go](https://www.hackerrank.com/domains/go)（语法专项）

**进阶**
- 《Go 语言实战》《Go 语言高级编程》（chai2010，在线免费）
- 《100 Go Mistakes》（避坑圣经，英文，有中文翻译版）
- [Go 夜读](https://github.com/talk-go/night)（源码阅读）
- [go-zero 文档](https://go-zero.dev/)、[Kratos 文档](https://go-kratos.dev/)

---

## 8. 30 天后你应该拥有

1. GitHub 上 3 个能跑的项目（爬虫、短链、OCR 平台），有 README、有测试、有压测数据。
2. 一份自己的 Go 知识库（30 篇日笔记）。
3. 一份架构设计文档（含架构图、容量估算、取舍说明）。
4. 能在 10 分钟内讲清一个项目的**背景、方案、权衡、结果**。

> 架构师不是"知道得多"，而是"能在约束下做对取舍，并让团队照着落地"。30 天能给你前者，后者要在真实项目里练。
