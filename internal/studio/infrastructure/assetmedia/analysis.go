package assetmedia

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"mime"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const (
	maxImageBytes  = 64 << 20
	maxImagePixels = 40_000_000
	maxSampleSide  = 256
	maxColors      = 10
	maxVideoBytes  = 64 << 20
)

type Color struct {
	Hex   string  `json:"hex"`
	Ratio float64 `json:"ratio"`
}

type ImageResult struct {
	Width  int
	Height int
	Colors []Color
}

type FileInfo struct {
	MIMEType string
	Format   string
	SHA256   string
	WidthPx  int
	HeightPx int
}

type VideoResult struct {
	Width        int
	Height       int
	Colors       []Color
	SamplePoints []float64
}

type colorBin struct {
	r, g, b float64
	weight  float64
	lab     [3]float64
}

func InspectFile(ctx context.Context, content []byte, _ string) (FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return FileInfo{}, err
	}
	info := FileInfo{MIMEType: "application/octet-stream", Format: "unknown", SHA256: fmt.Sprintf("%x", sha256.Sum256(content))}
	if len(content) == 0 {
		return info, nil
	}
	mediaType, _, err := mime.ParseMediaType(http.DetectContentType(content))
	if err != nil {
		return FileInfo{}, err
	}
	if looksLikeXML(content) {
		_, width, height, err := parseSVG(content)
		if err == nil {
			info.MIMEType, info.Format = "image/svg+xml", "svg"
			info.WidthPx, info.HeightPx = width, height
		}
		return info, nil
	}
	switch mediaType {
	case "image/png":
		info.Format = "png"
	case "image/jpeg":
		info.Format = "jpeg"
	case "image/gif":
		info.Format = "gif"
	case "image/webp":
		info.Format = "webp"
	case "video/mp4":
		info.Format = "mp4"
	case "video/webm":
		info.Format = "webm"
	case "application/pdf":
		info.Format = "pdf"
	case "text/plain":
		info.Format = "txt"
	case "application/zip":
		info.Format = "zip"
	}
	if info.Format == "unknown" {
		return info, nil
	}
	info.MIMEType = mediaType
	if mediaType == "image/webp" {
		info.WidthPx, info.HeightPx, err = probeDimensions(ctx, content)
	} else if mediaType == "image/png" || mediaType == "image/jpeg" || mediaType == "image/gif" {
		var config image.Config
		config, _, err = image.DecodeConfig(bytes.NewReader(content))
		info.WidthPx, info.HeightPx = config.Width, config.Height
	}
	if err != nil {
		return FileInfo{}, fmt.Errorf("读取文件尺寸失败: %w", err)
	}
	return info, nil
}

func AnalyzeImage(ctx context.Context, input io.Reader) (ImageResult, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxImageBytes+1))
	if err != nil {
		return ImageResult{}, fmt.Errorf("读取图片失败: %w", err)
	}
	if len(data) > maxImageBytes {
		return ImageResult{}, fmt.Errorf("图片超过分析大小限制")
	}
	mediaType, _, err := mime.ParseMediaType(http.DetectContentType(data))
	if err != nil {
		return ImageResult{}, err
	}
	if looksLikeXML(data) {
		icon, width, height, err := parseSVG(data)
		if err != nil {
			return ImageResult{}, fmt.Errorf("解析 SVG 失败: %w", err)
		}
		img, err := renderSVG(ctx, icon, width, height)
		if err != nil {
			return ImageResult{}, err
		}
		colors, err := analyzePixels(ctx, img)
		if err != nil {
			return ImageResult{}, err
		}
		return ImageResult{Width: width, Height: height, Colors: colors}, nil
	}
	var width, height int
	if mediaType == "image/webp" {
		width, height, err = probeDimensions(ctx, data)
	} else {
		var config image.Config
		config, _, err = image.DecodeConfig(bytes.NewReader(data))
		width, height = config.Width, config.Height
	}
	if err != nil {
		return ImageResult{}, fmt.Errorf("读取图片尺寸失败: %w", err)
	}
	if width <= 0 || height <= 0 || int64(width)*int64(height) > maxImagePixels {
		return ImageResult{}, fmt.Errorf("图片尺寸超过分析限制")
	}
	var img image.Image
	if mediaType == "image/webp" {
		processCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		command := exec.CommandContext(processCtx, "ffmpeg", "-nostdin", "-v", "error", "-i", "pipe:0", "-frames:v", "1", "-vf", "scale=256:256:force_original_aspect_ratio=decrease", "-f", "image2pipe", "-vcodec", "png", "pipe:1")
		command.Stdin = bytes.NewReader(data)
		var output []byte
		output, err = command.Output()
		if err == nil {
			img, err = png.Decode(bytes.NewReader(output))
		}
	} else {
		img, _, err = image.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return ImageResult{}, fmt.Errorf("解码图片失败: %w", err)
	}
	colors, err := analyzePixels(ctx, img)
	if err != nil {
		return ImageResult{}, err
	}
	return ImageResult{Width: width, Height: height, Colors: colors}, nil
}

func looksLikeXML(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && trimmed[0] == '<'
}

func parseSVG(data []byte) (*oksvg.SvgIcon, int, int, error) {
	document := etree.NewDocument()
	if err := document.ReadFromBytes(data); err != nil {
		return nil, 0, 0, err
	}
	root := document.Root()
	if root == nil || root.Tag != "svg" {
		return nil, 0, 0, fmt.Errorf("文件根元素不是 SVG")
	}
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, err
	}
	if !validSVGSize(icon.ViewBox.W) || !validSVGSize(icon.ViewBox.H) {
		return nil, 0, 0, fmt.Errorf("SVG 画布尺寸无效")
	}
	width := svgDimension(root.SelectAttrValue("width", ""))
	height := svgDimension(root.SelectAttrValue("height", ""))
	if width == 0 || height == 0 {
		return icon, 0, 0, nil
	}
	return icon, width, height, nil
}

func validSVGSize(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func svgDimension(raw string) int {
	raw = strings.TrimSpace(raw)
	factor := 1.0
	for _, unit := range []struct {
		suffix string
		factor float64
	}{
		{"px", 1}, {"in", 96}, {"cm", 96 / 2.54}, {"mm", 96 / 25.4},
		{"pt", 96 / 72}, {"pc", 16}, {"q", 96 / 101.6},
	} {
		if strings.HasSuffix(raw, unit.suffix) {
			raw = strings.TrimSpace(strings.TrimSuffix(raw, unit.suffix))
			factor = unit.factor
			break
		}
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || !validSVGSize(value) || value*factor > float64(math.MaxInt) {
		return 0
	}
	return int(math.Round(value * factor))
}

func renderSVG(ctx context.Context, icon *oksvg.SvgIcon, width, height int) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	canvasWidth, canvasHeight := icon.ViewBox.W, icon.ViewBox.H
	if width > 0 && height > 0 {
		if int64(width) > maxImagePixels/int64(height) {
			return nil, fmt.Errorf("图片尺寸超过分析限制")
		}
		canvasWidth, canvasHeight = float64(width), float64(height)
	}
	if !validSVGSize(canvasWidth) || !validSVGSize(canvasHeight) {
		return nil, fmt.Errorf("SVG 画布尺寸无效")
	}
	sampleWidth, sampleHeight := maxSampleSide, maxSampleSide
	if canvasWidth > canvasHeight {
		sampleHeight = max(1, int(math.Round(maxSampleSide*canvasHeight/canvasWidth)))
	} else {
		sampleWidth = max(1, int(math.Round(maxSampleSide*canvasWidth/canvasHeight)))
	}
	img := image.NewRGBA(image.Rect(0, 0, sampleWidth, sampleHeight))
	scanner := rasterx.NewScannerGV(sampleWidth, sampleHeight, img, img.Bounds())
	raster := rasterx.NewDasher(sampleWidth, sampleHeight, scanner)
	icon.SetTarget(0, 0, float64(sampleWidth), float64(sampleHeight))
	icon.Draw(raster, 1)
	return img, ctx.Err()
}

func AnalyzeVideo(ctx context.Context, input io.Reader) (VideoResult, error) {
	data, err := io.ReadAll(io.LimitReader(input, maxVideoBytes+1))
	if err != nil {
		return VideoResult{}, fmt.Errorf("读取视频失败: %w", err)
	}
	if len(data) > maxVideoBytes {
		return VideoResult{}, fmt.Errorf("视频超过分析读取上限")
	}
	return AnalyzeVideoStream(ctx, bytes.NewReader(data), bytes.NewReader(data))
}

func AnalyzeVideoStream(ctx context.Context, probeInput, frameInput io.Reader) (VideoResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return VideoResult{}, err
	}
	remaining := int64(maxVideoBytes)
	width, height, err := probeDimensionsFromReader(ctx, &budgetReader{reader: probeInput, remaining: &remaining})
	if err != nil {
		return VideoResult{}, fmt.Errorf("读取视频尺寸失败: %w", err)
	}
	command := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-v", "error", "-probesize", "1048576", "-analyzeduration", "1000000", "-i", "pipe:0", "-t", "10", "-vf", "fps=1/3,scale=256:256:force_original_aspect_ratio=decrease", "-frames:v", "4", "-f", "image2pipe", "-vcodec", "png", "pipe:1")
	command.Stdin = &budgetReader{reader: frameInput, remaining: &remaining}
	output, err := command.Output()
	if err != nil {
		return VideoResult{}, fmt.Errorf("分析视频画面失败: %w", err)
	}
	reader := bytes.NewReader(output)
	frames := make([]image.Image, 0, 4)
	for reader.Len() > 0 {
		frame, err := png.Decode(reader)
		if err != nil {
			return VideoResult{}, fmt.Errorf("读取视频画面失败: %w", err)
		}
		frames = append(frames, frame)
	}
	if len(frames) == 0 {
		return VideoResult{}, fmt.Errorf("视频没有可分析的画面")
	}
	sampleWidth, sampleHeight := frames[0].Bounds().Dx(), frames[0].Bounds().Dy()
	combined := image.NewNRGBA(image.Rect(0, 0, sampleWidth*len(frames), sampleHeight))
	for i, frame := range frames {
		if frame.Bounds().Dx() != sampleWidth || frame.Bounds().Dy() != sampleHeight {
			return VideoResult{}, fmt.Errorf("视频画面尺寸不一致")
		}
		draw.Draw(combined, image.Rect(i*sampleWidth, 0, (i+1)*sampleWidth, sampleHeight), frame, frame.Bounds().Min, draw.Src)
	}
	colors, err := analyzePixels(ctx, combined)
	if err != nil {
		return VideoResult{}, err
	}
	result := VideoResult{Width: width, Height: height, Colors: colors, SamplePoints: make([]float64, len(frames))}
	for i := range frames {
		result.SamplePoints[i] = float64(i * 3)
	}
	return result, nil
}

type budgetReader struct {
	reader    io.Reader
	remaining *int64
}

func (reader *budgetReader) Read(buffer []byte) (int, error) {
	if *reader.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(buffer)) > *reader.remaining {
		buffer = buffer[:*reader.remaining]
	}
	n, err := reader.reader.Read(buffer)
	*reader.remaining -= int64(n)
	return n, err
}

func probeDimensions(ctx context.Context, data []byte) (int, int, error) {
	return probeDimensionsFromReader(ctx, bytes.NewReader(data))
}

func probeDimensionsFromReader(ctx context.Context, input io.Reader) (int, int, error) {
	processCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	command := exec.CommandContext(processCtx, "ffprobe", "-v", "error", "-probesize", "1048576", "-analyzeduration", "1000000", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "json", "-i", "pipe:0")
	command.Stdin = input
	output, err := command.Output()
	if err != nil {
		return 0, 0, err
	}
	var result struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, 0, err
	}
	if len(result.Streams) == 0 || result.Streams[0].Width <= 0 || result.Streams[0].Height <= 0 {
		return 0, 0, fmt.Errorf("没有可识别的画面尺寸")
	}
	return result.Streams[0].Width, result.Streams[0].Height, nil
}

func analyzePixels(ctx context.Context, img image.Image) ([]Color, error) {
	bounds := img.Bounds()
	step := max(1, (bounds.Dx()+maxSampleSide-1)/maxSampleSide, (bounds.Dy()+maxSampleSide-1)/maxSampleSide)
	bins := make(map[uint16]*colorBin)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			pixel := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if pixel.A == 0 {
				continue
			}
			weight := float64(pixel.A) / 255
			key := uint16(pixel.R>>3)<<10 | uint16(pixel.G>>3)<<5 | uint16(pixel.B>>3)
			bin := bins[key]
			if bin == nil {
				bin = &colorBin{}
				bins[key] = bin
			}
			bin.r += float64(pixel.R) * weight
			bin.g += float64(pixel.G) * weight
			bin.b += float64(pixel.B) * weight
			bin.weight += weight
		}
	}
	if len(bins) == 0 {
		return nil, nil
	}
	values := make([]colorBin, 0, len(bins))
	for _, bin := range bins {
		bin.r /= bin.weight
		bin.g /= bin.weight
		bin.b /= bin.weight
		bin.lab = toLab(bin.r, bin.g, bin.b)
		values = append(values, *bin)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].weight != values[j].weight {
			return values[i].weight > values[j].weight
		}
		return values[i].lab[0] < values[j].lab[0]
	})
	count := min(maxColors, len(values))
	centers := make([]colorBin, 0, count)
	centers = append(centers, values[0])
	for len(centers) < count {
		bestIndex := 0
		bestScore := -1.0
		for i, candidate := range values {
			distance := math.MaxFloat64
			for _, center := range centers {
				distance = min(distance, labDistance(candidate.lab, center.lab))
			}
			score := candidate.weight * distance
			if score > bestScore {
				bestScore, bestIndex = score, i
			}
		}
		centers = append(centers, values[bestIndex])
	}
	for range 8 {
		next := make([]colorBin, count)
		for _, candidate := range values {
			index := nearestCenter(candidate.lab, centers)
			next[index].r += candidate.r * candidate.weight
			next[index].g += candidate.g * candidate.weight
			next[index].b += candidate.b * candidate.weight
			next[index].weight += candidate.weight
		}
		for i := range next {
			if next[i].weight == 0 {
				next[i] = centers[i]
				continue
			}
			next[i].r /= next[i].weight
			next[i].g /= next[i].weight
			next[i].b /= next[i].weight
			next[i].lab = toLab(next[i].r, next[i].g, next[i].b)
		}
		centers = next
	}
	total := 0.0
	for _, center := range centers {
		total += center.weight
	}
	result := make([]Color, 0, count)
	for _, center := range centers {
		if center.weight == 0 {
			continue
		}
		result = append(result, Color{
			Hex:   fmt.Sprintf("#%02X%02X%02X", uint8(math.Round(center.r)), uint8(math.Round(center.g)), uint8(math.Round(center.b))),
			Ratio: center.weight / total,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Ratio != result[j].Ratio {
			return result[i].Ratio > result[j].Ratio
		}
		return result[i].Hex < result[j].Hex
	})
	return result, nil
}

func nearestCenter(value [3]float64, centers []colorBin) int {
	index := 0
	distance := math.MaxFloat64
	for i, center := range centers {
		candidate := labDistance(value, center.lab)
		if candidate < distance {
			index, distance = i, candidate
		}
	}
	return index
}

func labDistance(a, b [3]float64) float64 {
	return math.Pow(a[0]-b[0], 2) + math.Pow(a[1]-b[1], 2) + math.Pow(a[2]-b[2], 2)
}

func toLab(r, g, b float64) [3]float64 {
	linear := func(value float64) float64 {
		value /= 255
		if value <= 0.04045 {
			return value / 12.92
		}
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	r, g, b = linear(r), linear(g), linear(b)
	x := (0.4124564*r + 0.3575761*g + 0.1804375*b) / 0.95047
	y := (0.2126729*r + 0.7151522*g + 0.0721750*b)
	z := (0.0193339*r + 0.1191920*g + 0.9503041*b) / 1.08883
	f := func(value float64) float64 {
		if value > 0.008856451679 {
			return math.Cbrt(value)
		}
		return value/0.128418549346 + 4.0/29
	}
	x, y, z = f(x), f(y), f(z)
	return [3]float64{116*y - 16, 500 * (x - y), 200 * (y - z)}
}
