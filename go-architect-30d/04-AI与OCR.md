# 04 · Go 在 AI 与 OCR 领域的应用（D22–D26）

> 先明确一个事实：**Go 不是训练语言**。模型训练、微调在 Python 生态。
> Go 的战场是：**推理服务、AI 网关、OCR 流水线、RAG 检索、Agent 编排、高并发接入**。
> 这一章教你把 AI 能力"工程化"——这才是公司真正付钱的部分。

---

## 1. Go 在 AI 技术栈中的位置

```
┌──────────────────────────────────────────────────────────────┐
│  业务系统（Go）：订单、文档、客服、审核                          │
├──────────────────────────────────────────────────────────────┤
│  AI 网关（Go）：鉴权、限流、路由、重试、计费、日志、降级          │
├──────────────────────────────────────────────────────────────┤
│  编排层（Go）：RAG、Agent、工具调用、任务队列、结果结构化         │
├──────────────────────────────────────────────────────────────┤
│  推理层：PaddleOCR(Python) / ONNX Runtime / vLLM / 云 API      │
├──────────────────────────────────────────────────────────────┤
│  训练层（Python）：PyTorch、微调、数据标注 —— Go 不参与          │
└──────────────────────────────────────────────────────────────┘
```

Go 的三个天然优势：

1. **高并发**：一个进程扛几千路 OCR 请求，协程 + 连接池是强项。
2. **部署简单**：单二进制，容器镜像 10MB，边缘设备也能跑。
3. **工程化**：限流、熔断、重试、可观测，生态成熟。

### 1.1 Go 做 AI 网关：最现实的岗位需求

公司里最常见的 Go + AI 结合点，不是训练模型，而是做一层**统一入口**：

| 能力 | 实现要点 |
|---|---|
| 多供应商路由 | 按模型/成本/可用性分流（OpenAI / DeepSeek / 通义 / 本地 vLLM） |
| 密钥托管 | 业务方不接触上游 Key，网关统一鉴权、配额 |
| 限流与计费 | 按 token/请求数限流，记录 usage 做账单 |
| 流式透传 | SSE 边收边转，注意 flush 与背压 |
| 降级重试 | 上游 5xx/超时切换备用模型，非幂等请求不重试 |
| 内容安全 | 敏感词、Prompt 注入检测、输出审核 |
| 可观测 | 每次调用的模型、耗时、token、成本、缓存命中 |

```go
// 极简路由示例
type Route struct {
	Name     string
	BaseURL  string
	Model    string
	Weight   int
	Fallback bool
}

func (g *Gateway) Pick(ctx context.Context) []Route {
	// 主路由 + 兜底路由，按权重随机
}
```

> 面试表达："我做的不是调 API，而是把不可靠、昂贵、多家的模型能力，封装成稳定、可计量、可降级的内部服务。"——这句话就是 AI 工程岗的核心。

---

## 2. OCR 技术全景

### 2.1 三代技术

| 代际 | 代表 | 原理 | 中文精度 | 适用 |
|---|---|---|---|---|
| 传统 | Tesseract 4/5 | LSTM + 传统图像处理 | 一般（60~80%） | 英文文档、票据、离线轻量 |
| 深度学习 | PaddleOCR、RapidOCR | DB 检测 + CRNN/SVTR 识别 | 高（90%+） | 中文场景主力 |
| 多模态大模型 | Qwen-VL、GPT-4o、Gemini | 端到端读图 | 很高（但慢、贵） | 复杂版面、手写、理解型任务 |

**一句话选型**：结构化票据/证件 → PaddleOCR；复杂版面理解/手写 → VLM；英文简单文档/离线 → Tesseract；不想运维 → 云 API。

再补两个高频概念（做合同、报表必用）：

- **版面分析（Layout Analysis）**：先把页面切成标题、正文、表格、图片等区域，再对每个区域分别识别。PaddleOCR 的 PP-Structure 就是干这个的，"先分区、再 OCR"比整页识别准得多。
- **表格识别**：检测表格线与单元格 → 逐格 OCR → 还原成 HTML/Excel。单元格合并、无线表格是主要难点，复杂场景直接上 VLM。

### 2.2 四条落地路线对比

| 路线 | 精度 | 速度 | 成本 | 部署复杂度 | 推荐场景 |
|---|---|---|---|---|---|
| **A. 云 API**（腾讯/阿里/百度） | 高 | 网络延迟 200~800ms | 按次计费 | 极低 | 快速上线、量小、不想运维 |
| **B. Tesseract**（gosseract） | 中 | 本地 50~300ms | 免费 | 中（CGO + 语言包） | 离线、英文、嵌入式 |
| **C. PaddleOCR 服务化** | 高 | 本地 100~500ms（GPU 更快） | 服务器成本 | 中（Python 容器） | **中文业务主力，推荐** |
| **D. ONNX Runtime**（Go 绑定） | 取决于模型 | 本地 30~200ms | 免费 | 高（CGO + 动态库） | 自研模型、边缘、极致性能 |

---

## 3. 路线 A：云 OCR（最快落地）

以腾讯云为例（阿里云、百度智能云的调用方式几乎一样）：

```go
package ocr

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ocr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ocr/v20181119"
)

type Client struct {
	cli *ocr.Client
}

func New(secretID, secretKey string) (*Client, error) {
	cred := common.NewCredential(secretID, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ocr.tencentcloudapi.com"
	cpf.HttpProfile.ReqTimeout = 10
	c, err := ocr.NewClient(cred, "", cpf)
	if err != nil {
		return nil, err
	}
	return &Client{cli: c}, nil
}

func (c *Client) RecognizeFile(ctx context.Context, path string) ([]string, error) {
	img, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	req := ocr.NewGeneralBasicOCRRequest()
	req.ImageBase64 = common.StringPtr(base64.StdEncoding.EncodeToString(img))
	req.LanguageType = common.StringPtr("zh")

	resp, err := c.cli.GeneralBasicOCRWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ocr request: %w", err)
	}

	var lines []string
	for _, item := range resp.Response.TextDetections {
		lines = append(lines, *item.DetectedText)
	}
	return lines, nil
}
```

专用票据接口直接用 `RecognizeGeneralInvoice` / `RecognizeIDCard` / `RecognizeBankCard`，比通用识别准很多。

**工程要点**

- 密钥**绝不进代码**：环境变量 / KMS / 密钥管理服务。
- 超时 + 重试（指数退避，只对 5xx/超时重试）。
- 并发控制：SDK 默认 3 秒超时，QPS 高时用 worker pool 限制并发（云厂商有 QPS 上限）。
- 结果缓存：同一文件 SHA256 做 key，重复识别直接命中。

---

## 4. 路线 B：Tesseract（离线 / CGO）

```go
import "github.com/otiai10/gosseract/v2"

func TessOCR(path string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()

	client.SetImage(path)
	client.SetLanguage("chi_sim", "eng")   // 需要安装对应语言包
	client.SetPageSegMode(gosseract.PSM_AUTO)

	return client.Text()
}
```

环境要求：

- Windows：安装 Tesseract，把 `.dll` 放到 PATH，`CGO_ENABLED=1`。
- Linux：`apt install tesseract-ocr tesseract-ocr-chi-sim libtesseract-dev`。
- 训练数据放在 `TESSDATA_PREFIX`。

优缺点：免费、离线、英文很准；**中文精度一般**，版式复杂时错行严重。适合"扫描件文字提取"而不是"票据结构化"。

> 交叉编译会因 CGO 失败。要么在目标平台构建，要么容器里构建。

---

## 5. 路线 C：PaddleOCR 服务化（推荐主力方案）

思路：**Python 跑模型，Go 做业务**。两者通过 HTTP/gRPC 解耦。

### 5.1 Python 侧：包一层 FastAPI（约 40 行）

```python
# ocr_server.py
import io, base64
from fastapi import FastAPI, UploadFile, File
from PIL import Image
import numpy as np
from paddleocr import PaddleOCR

app = FastAPI()
ocr = PaddleOCR(use_angle_cls=True, lang="ch", show_log=False)

@app.post("/ocr")
async def recognize(file: UploadFile = File(...)):
    img = Image.open(io.BytesIO(await file.read())).convert("RGB")
    result = ocr.ocr(np.array(img), cls=True)
    lines = []
    for page in result:
        for box, (text, score) in page or []:
            lines.append({"text": text, "score": round(score, 4),
                          "box": [[int(x), int(y)] for x, y in box]})
    return {"lines": lines, "count": len(lines)}
```

```dockerfile
FROM paddlepaddle/paddle:2.6.1-gpu-cuda11.8-cudnn8.6-trt8.5
RUN pip install paddleocr fastapi uvicorn python-multipart
COPY ocr_server.py /app/
WORKDIR /app
CMD ["uvicorn", "ocr_server:app", "--host", "0.0.0.0", "--port", "8000"]
```

### 5.2 Go 侧：客户端（带重试、超时、并发限制）

```go
type OCRClient struct {
	baseURL string
	http    *http.Client
	sem     chan struct{}
}

func NewOCRClient(baseURL string, concurrency int) *OCRClient {
	return &OCRClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
		sem:     make(chan struct{}, concurrency),
	}
}

type Line struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

type ocrResp struct {
	Lines []Line `json:"lines"`
}

func (c *OCRClient) Recognize(ctx context.Context, filename string, data []byte) ([]Line, error) {
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(data); err != nil {
		return nil, err
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/ocr", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ocr http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("ocr status=%d body=%s", resp.StatusCode, body)
	}

	var out ocrResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return out.Lines, nil
}
```

### 5.3 部署形态

| 方式 | 说明 |
|---|---|
| 单机 Docker | 小流量，Go 服务通过 `http://ocr:8000` 调用 |
| K8s + HPA | 按队列长度扩容，GPU 节点池 |
| Triton Inference Server | 多模型统一托管，支持 gRPC |
| RapidOCR（ONNX） | 纯 CPU、体积小，精度略低但部署极简 |

> PaddleOCR 官方没有长期维护的 HTTP 服务，社区常用 FastAPI 包装或 `paddleocr-json`（C++ 版，直接产出 JSON）。生产上更稳的做法是 Triton + ONNX 导出。

---

## 6. 路线 D：ONNX Runtime（Go 侧推理）

```go
import ort "github.com/yalue/onnxruntime_go"

func init() {
	ort.SetSharedLibraryPath(`C:\onnxruntime\onnxruntime.dll`) // 或 .so
	if err := ort.InitializeEnvironment(); err != nil {
		panic(err)
	}
}

func Infer(input []float32) ([]float32, error) {
	shape := ort.NewShape(1, 3, 224, 224)
	inputTensor, err := ort.NewTensor(shape, input)
	if err != nil {
		return nil, err
	}
	defer inputTensor.Destroy()

	outputTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 1000))
	if err != nil {
		return nil, err
	}
	defer outputTensor.Destroy()

	session, err := ort.NewAdvancedSession("model.onnx",
		[]string{"input"}, []string{"output"},
		[]ort.Value{inputTensor}, []ort.Value{outputTensor}, nil)
	if err != nil {
		return nil, err
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return nil, err
	}
	return outputTensor.GetData(), nil
}
```

限制与建议：

- 需要系统安装 ONNX Runtime 动态库（版本要与绑定库匹配）。
- CGO 打开，交叉编译麻烦。
- 适合把**检测/识别小模型**做到 Go 进程内，省一次网络往返；大模型仍建议独立服务。

---

## 7. 图像预处理：OCR 精度的 40% 在这里

OCR 不是"丢进去就完事"。做一个 `preprocess` 包：

```go
// 灰度 + Otsu 二值化 + 尺寸归一化
func Preprocess(src image.Image) *image.Gray {
	b := src.Bounds()
	gray := image.NewGray(b)
	draw.Draw(gray, b, src, b.Min, draw.Src)

	th := otsuThreshold(gray)
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if gray.GrayAt(x, y).Y > th {
				out.SetGray(x, y, color.Gray{Y: 255})
			} else {
				out.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}
	return out
}

// 大津法：最大化类间方差
func otsuThreshold(img *image.Gray) uint8 {
	var hist [256]int
	b := img.Bounds()
	total := b.Dx() * b.Dy()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			hist[img.GrayAt(x, y).Y]++
		}
	}
	var sum float64
	for i := 0; i < 256; i++ {
		sum += float64(i * hist[i])
	}
	var sumB, wB float64
	maxVar, threshold := 0.0, 0
	for i := 0; i < 256; i++ {
		wB += float64(hist[i])
		if wB == 0 {
			continue
		}
		wF := float64(total) - wB
		if wF == 0 {
			break
		}
		sumB += float64(i * hist[i])
		mB := sumB / wB
		mF := (sum - sumB) / wF
		variance := wB * wF * (mB - mF) * (mB - mF)
		if variance > maxVar {
			maxVar = variance
			threshold = i
		}
	}
	return uint8(threshold)
}
```

完整预处理链路：

```
原图 → 解码 → EXIF 方向校正 → 灰度 → 去噪(高斯/中值) → 二值化(Otsu/自适应)
     → 倾斜校正(霍夫变换/最小外接矩形) → 裁剪 ROI → 缩放(高 32px 倍数) → OCR
```

常用库：`golang.org/x/image`（draw、缩放）、`github.com/disintegration/imaging`（裁剪/旋转）、`github.com/anthonynsimon/bild`（图像处理全家桶）。

PDF 处理：

- `pdftoppm`（poppler）转 PNG，Go 用 `os/exec` 调。
- 纯 Go：`github.com/gen2brain/go-fitz`（MuPDF 绑定，CGO）。
- 扫描版 PDF 先转图再 OCR；文字版 PDF 直接抽文本（`github.com/ledongthuc/pdf`）。

---

## 8. OCR 工程化：从"能识别"到"能上线"

### 8.1 异步任务架构

```
上传 → 存对象存储(OSS/S3) → 发 MQ → 消费 worker → OCR → LLM 结构化 → 落库 → 回调/轮询
```

为什么要异步：OCR 单张 200ms~2s，同步接口会拖垮连接池；异步还能削峰、重试、限流。

```go
// 任务状态机：pending → processing → success / failed / need_review
type Task struct {
	ID        string
	Status    string
	Retries   int
	ResultURL string
}
```

### 8.2 结果结构化：正则 + LLM 双保险

```go
// 第一层：正则抽关键字段（快、免费）
var (
	invoiceNo = regexp.MustCompile(`发票号码[:：]?\s*(\d{8,20})`)
	amount    = regexp.MustCompile(`价税合计.*?[¥￥]?\s*([\d,]+\.\d{2})`)
)

// 第二层：LLM 兜底（处理版式变化）
prompt := fmt.Sprintf(`从下面 OCR 文本中提取 JSON：
{"invoice_no":"","date":"","amount":0,"seller":""}
只输出 JSON。文本：
%s`, strings.Join(lines, "\n"))
```

### 8.3 质量保障

| 手段 | 说明 |
|---|---|
| 置信度阈值 | 平均 score < 0.8 转人工复核 |
| 字段校验 | 金额格式、日期合法性、税号校验位 |
| 交叉校验 | 金额合计 = 各项之和 |
| 影子模式 | 新模型先跑 5% 流量对比 |
| 人工抽检 | 每天抽 1% 复核，统计准确率 |

---

## 9. LLM 接入（D25）

### 9.1 统一用 OpenAI 兼容协议

DeepSeek、通义千问、Kimi、vLLM、Ollama、OneAPI 都兼容 `/v1/chat/completions`。写一套代码，换 `base_url` 和 `model` 即可。

```go
type Message struct {
	Role    string `json:"role"`    // system / user / assistant
	Content string `json:"content"`
}

type ChatRequest struct {
	Model          string            `json:"model"`
	Messages       []Message         `json:"messages"`
	Temperature    float64           `json:"temperature,omitempty"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	Stream         bool              `json:"stream,omitempty"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

func Chat(ctx context.Context, baseURL, apiKey string, req ChatRequest) (string, error) {
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return "", fmt.Errorf("llm status=%d body=%s", resp.StatusCode, b)
	}

	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("empty choices")
	}
	return out.Choices[0].Message.Content, nil
}
```

生产别自己手写：用 `github.com/openai/openai-go`（官方 Go SDK）或 `github.com/sashabaranov/go-openai`。

### 9.2 流式输出（SSE）—— 用户体验的关键

```go
func StreamChat(ctx context.Context, baseURL, apiKey, model, prompt string,
	onDelta func(string)) error {

	body, _ := json.Marshal(ChatRequest{
		Model:    model,
		Messages: []Message{{Role: "user", Content: prompt}},
		Stream:   true,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := (&http.Client{Timeout: 0}).Do(req) // 流式不要设总超时，用 ctx 控制
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onDelta(chunk.Choices[0].Delta.Content)
		}
	}
	return sc.Err()
}
```

在 Gin 里把流式内容转发给前端：

```go
func ChatHandler(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok { c.Status(500); return }

	err := llm.StreamChat(c.Request.Context(), baseURL, apiKey, model, prompt,
		func(delta string) {
			fmt.Fprintf(c.Writer, "data: %s\n\n", delta)
			flusher.Flush()
		})
	if err != nil {
		fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", err)
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
}
```

要点：**客户端断开要能感知**（`c.Request.Context()` 会取消），否则模型还在烧 token。

### 9.3 结构化输出（让 LLM 返回可靠 JSON）

三种做法，可靠性递增：

1. Prompt 里给 schema + "只输出 JSON"（简单，但会跑偏）。
2. `response_format: {"type": "json_object"}`（OpenAI/DeepSeek 支持）。
3. Function calling / tool use：把 schema 声明成工具，模型返回结构化参数。

```go
req := ChatRequest{
	Model: "deepseek-chat",
	Messages: []Message{{Role: "user", Content: prompt}},
	ResponseFormat: map[string]string{"type": "json_object"},
}
```

拿到结果后**必须校验**：`json.Valid` + 结构体反序列化 + 业务校验，失败重试一次（带错误信息）。

### 9.4 成本与稳定性

| 问题 | 对策 |
|---|---|
| token 费用高 | 缓存相同请求（prompt hash）、小模型做分类、大模型只做兜底 |
| 超时/限流 | 超时 30~60s、指数退避重试、多供应商路由 |
| 结果不稳定 | temperature 0.1~0.3、few-shot、输出校验 + 重试 |
| 长文本 | 分段总结（map-reduce）、RAG 只喂相关片段 |
| 隐私 | 脱敏后再发、私有化部署（vLLM + 本地模型） |

---

## 10. RAG：让 LLM 回答公司内部文档（D26）

### 10.1 流程

```
离线：文档 → 解析(pdf/docx) → 清洗 → 切分(500~1000 字, 重叠 10~20%)
     → embedding → 写入向量库

在线：问题 → embedding → 向量检索 TopK → (可选)重排 → 拼 Prompt → LLM → 答案 + 引用
```

### 10.2 切分策略

| 策略 | 说明 |
|---|---|
| 固定长度 + 重叠 | 最简单，通用 |
| 按语义/段落 | 按标题、空行切，保留上下文 |
| 递归切分 | 先按章节，再按段落，再按句子 |
| 父子块 | 小块检索，命中后返回父块给 LLM（效果好，推荐） |

### 10.3 向量库选型

| 方案 | 优点 | 缺点 |
|---|---|---|
| **pgvector** | 复用 PG，事务一致，运维简单 | 千万级以后性能下降 |
| **Milvus** | 专业、十亿级、多索引 | 组件多，运维重 |
| **Qdrant** | Rust 写的，过滤强、部署简单 | 生态比 Milvus 小 |
| **Redis Search** | 已有 Redis 直接用 | 功能相对少 |

小团队直接 **pgvector**：

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE doc_chunks (
  id        bigserial PRIMARY KEY,
  doc_id    bigint NOT NULL,
  content   text   NOT NULL,
  embedding vector(1024) NOT NULL
);
CREATE INDEX ON doc_chunks USING hnsw (embedding vector_cosine_ops);

-- 检索
SELECT content, 1 - (embedding <=> $1) AS score
FROM doc_chunks
ORDER BY embedding <=> $1
LIMIT 5;
```

### 10.4 Go 实现要点

- embedding 调用（OpenAI `/v1/embeddings`、BGE-M3 本地服务）。
- 批量写入（500~1000 条一批）。
- 混合检索：向量 + 关键词（BM25/全文索引），再融合排序（RRF）。
- 重排：`bge-reranker` 服务，Top50 → Top5。
- Prompt 里带引用编号，前端可跳转原文。

生态库：

- `github.com/tmc/langchaingo`：链式编排、文档加载、向量库适配（学习用）。
- `github.com/cloudwego/eino`：字节开源，Go 原生 LLM 应用框架，组件化编排。
- `github.com/mark3labs/mcp-go`：把内部系统包装成 MCP Server，给 AI 客户端调用。

---

## 11. 视觉大模型（VLM）会取代 OCR 吗？

**部分会。** 对比：

| 场景 | 传统 OCR | VLM |
|---|---|---|
| 标准发票/身份证 | ✅ 快、便宜、稳定 | 慢、贵，但版式变化时更强 |
| 复杂版面（合同、报表） | 需版面分析，工程复杂 | ✅ 直接理解结构 |
| 手写体 | 差 | ✅ 明显更好 |
| 表格抽取 | 需要表格识别模型 | ✅ 一句话搞定 |
| 高并发（1w QPS） | ✅ 便宜 | ❌ 成本扛不住 |

**务实方案：分层处理。**

```
简单票据 → PaddleOCR（90% 流量，低成本）
  └─ 置信度低/字段缺失 → VLM 兜底（10% 流量）
        └─ 仍失败 → 人工复核
```

这就是"AI 架构"的价值：**用工程手段平衡成本、延迟、准确率**。

---

## 12. 实战：发票识别服务（D27 项目骨架）

### 12.1 架构

```
┌────────┐   ┌─────────┐   ┌──────────┐   ┌──────────┐   ┌─────────┐
│ 上传API │──▶│ 对象存储 │──▶│  MQ(Kafka)│──▶│ OCR Worker│──▶│ LLM 抽取 │
└────────┘   └─────────┘   └──────────┘   └──────────┘   └────┬────┘
     │                                                        │
     │  轮询/回调                                              ▼
     └───────────────────────────────────────────────▶ ┌──────────┐
                                                       │ MySQL+Redis│
                                                       └──────────┘
```

### 12.2 核心表

```sql
CREATE TABLE invoice_tasks (
  id          bigserial PRIMARY KEY,
  file_key    varchar(255) NOT NULL,
  file_hash   char(64) NOT NULL,
  status      varchar(20) NOT NULL DEFAULT 'pending', -- pending/processing/success/failed/review
  retries     int NOT NULL DEFAULT 0,
  raw_text    text,
  result      jsonb,
  confidence  numeric(4,3),
  created_at  timestamptz DEFAULT now(),
  updated_at  timestamptz DEFAULT now()
);
CREATE UNIQUE INDEX idx_hash ON invoice_tasks(file_hash);  -- 幂等：同一文件不重复处理
```

### 12.3 幂等 + 重试

```go
func (w *Worker) Handle(ctx context.Context, msg Message) error {
	// 1. 幂等：已成功直接跳过
	task, err := w.repo.Get(ctx, msg.TaskID)
	if err != nil { return err }
	if task.Status == "success" { return nil }

	// 2. 置为处理中（乐观锁防并发重复消费）
	if err := w.repo.MarkProcessing(ctx, task.ID); err != nil { return nil }

	// 3. OCR
	lines, err := w.ocr.Recognize(ctx, task.FileKey)
	if err != nil { return w.retry(ctx, task, err) }

	// 4. LLM 结构化（失败降级为正则）
	inv, err := w.extract(ctx, lines)
	if err != nil {
		inv = w.extractByRegex(lines)
	}

	// 5. 落库 + 缓存
	return w.repo.SaveResult(ctx, task.ID, inv)
}
```

### 12.4 验收标准

- [ ] 同一张发票上传 3 次，只生成 1 条记录（幂等）
- [ ] OCR 服务挂掉时，任务进重试队列，恢复后自动补处理
- [ ] P99 端到端 < 5s，单机吞吐 > 20 张/秒（OCR 并发 8）
- [ ] 关键字段准确率 > 95%（100 张测试集）
- [ ] 有 `/metrics`、有任务看板、有失败告警

---

## 13. AI 工程避坑清单

| 坑 | 后果 | 方案 |
|---|---|---|
| 同步调 OCR/LLM | 接口超时、连接池打满 | 异步任务 + 状态轮询 |
| 不设超时 | 协程堆积 | 每层都设 context 超时 |
| 不限制并发 | 打爆下游/超配额 | 信号量/worker pool |
| 直接信任 LLM 输出 | 脏数据入库 | schema 校验 + 重试 + 人工复核 |
| Prompt 硬编码 | 改一次要发版 | Prompt 模板化，配置中心管理 |
| 不记录 token 消耗 | 账单失控 | 每次调用记录 usage |
| 全量文档塞 context | 又慢又贵 | RAG 检索 + 摘要 |
| 忽略数据隐私 | 合规风险 | 脱敏、私有化部署、审计日志 |
| 只测 happy path | 上线即翻车 | 造异常样本：模糊图、旋转、缺字段 |

---

## 14. 学习资源

**OCR**
- [PaddleOCR](https://github.com/PaddlePaddle/PaddleOCR)（必看，含部署文档）
- [RapidOCR](https://github.com/RapidAI/RapidOCR)（ONNX 版，部署最简）
- [Tesseract](https://github.com/tesseract-ocr/tesseract) + [gosseract](https://github.com/otiai10/gosseract)

**LLM / RAG**
- [OpenAI API 文档](https://platform.openai.com/docs)（协议事实标准）
- [langchaingo](https://github.com/tmc/langchaingo)
- [eino](https://github.com/cloudwego/eino)（字节，Go 原生）
- [pgvector](https://github.com/pgvector/pgvector)

**推理部署**
- [ONNX Runtime Go](https://github.com/yalue/onnxruntime_go)
- [Triton Inference Server](https://github.com/triton-inference-server/server)
- [vLLM](https://github.com/vllm-project/vllm)

---

**下一篇**：[05-架构师之路.md](05-架构师之路.md)
