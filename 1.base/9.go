package main

import (
	"fmt"
	"strings"
)

func ProcessLog(raw string) string {
	lines := strings.Split(raw, "\n")
	var messages []string

	for _, line := range lines {
		// 1. 去掉每行首尾空白
		line = strings.TrimSpace(line)
		// 2. 过滤空行
		if line == "" {
			continue
		}
		// 3. 只保留 [INFO] 和 [ERROR]
		if !strings.HasPrefix(line, "[INFO]") && !strings.HasPrefix(line, "[ERROR]") {
			continue
		}
		// 4. [ERROR] → [ALERT]
		line = strings.ReplaceAll(line, "[ERROR]", "[ALERT]")
		// 5. 日期 → TODAY
		line = strings.ReplaceAll(line, "2024-01-15", "TODAY")

		// 6. 提取消息部分：找到 "] TODAY " 后面的内容
		idx := strings.Index(line, "] TODAY ")
		if idx == -1 {
			continue
		}
		msg := line[idx+len("] TODAY "):]
		// 7. 转小写
		msg = strings.ToLower(msg)
		messages = append(messages, msg)
	}

	// 8. 用 " | " 拼接
	result := strings.Join(messages, " | ")

	// 9. 前后加 << >>
	var b strings.Builder
	b.WriteString("<<")
	b.WriteString(result)
	b.WriteString(">>")
	final := b.String()

	// 10. 验证
	fmt.Println("以<<开头:", strings.HasPrefix(final, "<<"))
	fmt.Println("以>>结尾:", strings.HasSuffix(final, ">>"))
	fmt.Println("包含connection timeout:", strings.Contains(final, "connection timeout"))

	return final
}

func main() {
	raw := `  [INFO] 2024-01-15 server started on port 8080  
[ERROR] 2024-01-15 connection timeout
  [WARN] 2024-01-15 disk usage 90%  
[INFO] 2024-01-15 request received from 192.168.1.1
[ERROR] 2024-01-15 disk full
[DEBUG] 2024-01-15 variable x = 42`

	fmt.Println(ProcessLog(raw))
}