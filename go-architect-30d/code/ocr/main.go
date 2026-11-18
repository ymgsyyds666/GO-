// Package main 演示 04-AI与OCR.md 中的图像预处理：灰度化 + Otsu 二值化。
// 运行：go run ./ocr
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

func check(cond bool, msg string) {
	if !cond {
		panic("断言失败: " + msg)
	}
}

// toGray 转灰度图（Go 标准库的 color.GrayModel 使用 299/587/114 加权）。
func toGray(src image.Image) *image.Gray {
	b := src.Bounds()
	g := image.NewGray(b)
	draw.Draw(g, b, src, b.Min, draw.Src)
	return g
}

// otsu 大津法：遍历阈值，取类间方差最大者。
func otsu(img *image.Gray) uint8 {
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

// binarize 按阈值二值化：亮于阈值 → 白（背景），否则 → 黑（前景/文字）。
func binarize(img *image.Gray, threshold uint8) *image.Gray {
	b := img.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.GrayAt(x, y).Y > threshold {
				out.SetGray(x, y, color.Gray{Y: 255})
			} else {
				out.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}
	return out
}

// foregroundRatio 统计黑色像素占比，用来判断二值化是否把文字和背景分开了。
func foregroundRatio(img *image.Gray) float64 {
	b := img.Bounds()
	total := b.Dx() * b.Dy()
	var black int
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.GrayAt(x, y).Y == 0 {
				black++
			}
		}
	}
	return float64(black) / float64(total)
}

func main() {
	// 1. 构造一张"浅灰背景 + 红色文字条 + 轻微噪点"的图片
	const w, h = 320, 120
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Bounds(), image.NewUniform(color.RGBA{238, 238, 238, 255}), image.Point{}, draw.Src)

	// 噪点：模拟扫描件的底噪
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x*7+y*13)%23 == 0 {
				c := src.RGBAAt(x, y)
				c.R -= 20
				c.G -= 20
				c.B -= 20
				src.SetRGBA(x, y, c)
			}
		}
	}

	// "文字"：几条红色横条
	red := image.NewUniform(color.RGBA{200, 20, 20, 255})
	for _, r := range []image.Rectangle{
		image.Rect(20, 20, 90, 40),
		image.Rect(105, 20, 200, 40),
		image.Rect(215, 20, 300, 40),
		image.Rect(20, 65, 160, 85),
	} {
		draw.Draw(src, r, red, image.Point{}, draw.Src)
	}

	// 2. 灰度化
	gray := toGray(src)

	// 3. Otsu 求阈值并二值化
	threshold := otsu(gray)
	bw := binarize(gray, threshold)

	ratio := foregroundRatio(bw)
	// 文字区域约占 3%~15%，若二值化失败（全黑/全白）这个断言会失败
	check(ratio > 0.02 && ratio < 0.3, "前景占比应在合理区间")
	check(threshold > 20 && threshold < 235, "阈值不应是极端值")

	fmt.Printf("图像尺寸: %dx%d\n", w, h)
	fmt.Printf("Otsu 阈值: %d\n", threshold)
	fmt.Printf("前景（文字）像素占比: %.2f%%\n", ratio*100)

	// 4. 保存结果，方便肉眼检查（写到工作目录下的 out/）
	if err := os.MkdirAll("out", 0o755); err != nil {
		panic(err)
	}
	outPath := filepath.Join("out", "preprocess.png")
	f, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	if err := png.Encode(f, bw); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}

	info, _ := os.Stat(outPath)
	fmt.Printf("二值化结果已保存: %s (%d bytes)\n", outPath, info.Size())
	fmt.Println("预处理链路: 解码 → 灰度 → 去噪 → 二值化 → 裁剪/纠偏 → 缩放 → OCR")
}
