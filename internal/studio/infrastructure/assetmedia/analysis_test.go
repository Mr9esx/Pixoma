package assetmedia_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os/exec"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/assetmedia"
)

func TestAnalyzeImageCountsVisiblePixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	img.Set(1, 0, color.NRGBA{R: 255, A: 255})
	img.Set(2, 0, color.NRGBA{B: 255, A: 128})
	img.Set(3, 0, color.NRGBA{G: 255, A: 0})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}

	result, err := assetmedia.AnalyzeImage(context.Background(), bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 4 || result.Height != 1 {
		t.Fatalf("dimensions = %dx%d", result.Width, result.Height)
	}
	if len(result.Colors) != 2 {
		t.Fatalf("colors = %#v", result.Colors)
	}
	want := map[string]float64{"#FF0000": 2.0 / 2.5, "#0000FF": 0.5 / 2.5}
	for _, got := range result.Colors {
		ratio, ok := want[got.Hex]
		if !ok || math.Abs(got.Ratio-ratio) > 0.01 {
			t.Fatalf("color = %#v", got)
		}
	}
}

func TestInspectFileUsesContentForImageFormatAndDimensions(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for _, test := range []struct {
		name   string
		format string
		mime   string
		encode func(*bytes.Buffer) error
	}{
		{"PNG", "png", "image/png", func(dst *bytes.Buffer) error { return png.Encode(dst, img) }},
		{"JPEG", "jpeg", "image/jpeg", func(dst *bytes.Buffer) error { return jpeg.Encode(dst, img, nil) }},
		{"GIF", "gif", "image/gif", func(dst *bytes.Buffer) error { return gif.Encode(dst, img, nil) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var encoded bytes.Buffer
			if err := test.encode(&encoded); err != nil {
				t.Fatal(err)
			}
			info, err := assetmedia.InspectFile(context.Background(), encoded.Bytes(), "image/webp")
			if err != nil {
				t.Fatal(err)
			}
			if info.Format != test.format || info.MIMEType != test.mime || info.WidthPx != 3 || info.HeightPx != 2 {
				t.Fatalf("info = %#v", info)
			}
			if info.SHA256 != fmt.Sprintf("%x", sha256.Sum256(encoded.Bytes())) {
				t.Fatalf("SHA256 = %s", info.SHA256)
			}
		})
	}
}

func TestInspectFileWebP(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg 未安装")
	}
	img := image.NewNRGBA(image.Rect(0, 0, 6, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("ffmpeg", "-v", "error", "-f", "image2pipe", "-i", "pipe:0", "-frames:v", "1", "-f", "webp", "pipe:1")
	command.Stdin = bytes.NewReader(pngData.Bytes())
	webpData, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	info, err := assetmedia.InspectFile(context.Background(), webpData, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "webp" || info.MIMEType != "image/webp" || info.WidthPx != 6 || info.HeightPx != 4 {
		t.Fatalf("info = %#v", info)
	}
	palette, err := assetmedia.AnalyzeImage(context.Background(), bytes.NewReader(webpData))
	if err != nil {
		t.Fatal(err)
	}
	if palette.Width != 6 || palette.Height != 4 || len(palette.Colors) == 0 {
		t.Fatalf("palette = %#v", palette)
	}
}

func TestInspectFileUnknownContent(t *testing.T) {
	content := []byte{0, 1, 2, 3, 4}
	info, err := assetmedia.InspectFile(context.Background(), content, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "unknown" || info.MIMEType != "application/octet-stream" || info.WidthPx != 0 || info.HeightPx != 0 {
		t.Fatalf("info = %#v", info)
	}
}

func TestSVGUsesDeclaredDimensionsAndRendersPalette(t *testing.T) {
	content := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100" viewBox="0 0 20 10"><rect width="10" height="10" fill="#ff0000"/><rect x="10" width="10" height="10" fill="#0000ff"/></svg>`)
	info, err := assetmedia.InspectFile(context.Background(), content, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if info.MIMEType != "image/svg+xml" || info.Format != "svg" || info.WidthPx != 200 || info.HeightPx != 100 {
		t.Fatalf("SVG info = %#v", info)
	}
	if info.SHA256 != fmt.Sprintf("%x", sha256.Sum256(content)) {
		t.Fatalf("SVG SHA256 = %s", info.SHA256)
	}
	result, err := assetmedia.AnalyzeImage(context.Background(), bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 200 || result.Height != 100 || len(result.Colors) != 2 {
		t.Fatalf("SVG palette = %#v", result)
	}
	colors := make(map[string]float64, len(result.Colors))
	for _, item := range result.Colors {
		colors[item.Hex] = item.Ratio
	}
	if math.Abs(colors["#FF0000"]-0.5) > 0.05 || math.Abs(colors["#0000FF"]-0.5) > 0.05 {
		t.Fatalf("SVG colors = %#v", result.Colors)
	}
}

func TestSVGWithoutDeclaredDimensionsHasUnknownSize(t *testing.T) {
	content := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 10"><rect width="20" height="10" fill="#00ff00"/></svg>`)
	info, err := assetmedia.InspectFile(context.Background(), content, "image/svg+xml")
	if err != nil {
		t.Fatal(err)
	}
	if info.MIMEType != "image/svg+xml" || info.Format != "svg" || info.WidthPx != 0 || info.HeightPx != 0 {
		t.Fatalf("SVG info = %#v", info)
	}
	result, err := assetmedia.AnalyzeImage(context.Background(), bytes.NewReader(content))
	if err != nil || len(result.Colors) != 1 || result.Colors[0].Hex != "#00FF00" {
		t.Fatalf("SVG palette = %#v, %v", result, err)
	}
}

func TestInvalidSVGRemainsUnknown(t *testing.T) {
	content := []byte(`<html><svg width="20" height="10"></svg></html>`)
	info, err := assetmedia.InspectFile(context.Background(), content, "image/svg+xml")
	if err != nil {
		t.Fatal(err)
	}
	if info.MIMEType != "application/octet-stream" || info.Format != "unknown" {
		t.Fatalf("invalid SVG info = %#v", info)
	}
}

func TestInspectFileMP4(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg 未安装")
	}
	command := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=red:s=16x16:d=1:r=1", "-c:v", "mpeg4", "-movflags", "frag_keyframe+empty_moov", "-f", "mp4", "pipe:1")
	content, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	info, err := assetmedia.InspectFile(context.Background(), content, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "mp4" || info.MIMEType != "video/mp4" {
		t.Fatalf("info = %#v", info)
	}
}

func TestInspectFileWebM(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg 未安装")
	}
	command := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=blue:s=16x16:d=1:r=1", "-c:v", "libvpx-vp9", "-f", "webm", "pipe:1")
	content, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	info, err := assetmedia.InspectFile(context.Background(), content, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if info.Format != "webm" || info.MIMEType != "video/webm" {
		t.Fatalf("WebM info = %#v", info)
	}
}

func TestAnalyzeVideoReadsOnlyOpeningSegment(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg 未安装")
	}
	create := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=red:s=16x16:d=12:r=1", "-f", "lavfi", "-i", "color=c=blue:s=16x16:d=2:r=1", "-filter_complex", "[0:v][1:v]concat=n=2:v=1:a=0", "-c:v", "mpeg4", "-f", "matroska", "pipe:1")
	video, err := create.Output()
	if err != nil {
		t.Fatal(err)
	}

	result, err := assetmedia.AnalyzeVideo(context.Background(), bytes.NewReader(video))
	if err != nil {
		t.Fatal(err)
	}
	var probeRead, frameRead bytes.Buffer
	streamed, err := assetmedia.AnalyzeVideoStream(context.Background(),
		io.TeeReader(bytes.NewReader(video), &probeRead),
		io.TeeReader(bytes.NewReader(video), &frameRead))
	if err != nil {
		t.Fatal(err)
	}
	if streamed.Width != result.Width || streamed.Height != result.Height || len(streamed.Colors) == 0 {
		t.Fatalf("streamed result = %#v", streamed)
	}
	if probeRead.Len()+frameRead.Len() > 64<<20 {
		t.Fatalf("video read limit exceeded: probe=%d frame=%d", probeRead.Len(), frameRead.Len())
	}
	if len(result.SamplePoints) == 0 || len(result.SamplePoints) > 4 {
		t.Fatalf("sample points = %#v", result.SamplePoints)
	}
	if result.Width != 16 || result.Height != 16 {
		t.Fatalf("original dimensions = %dx%d", result.Width, result.Height)
	}
	for _, second := range result.SamplePoints {
		if second >= 10 {
			t.Fatalf("sampled after opening segment: %v", second)
		}
	}
	if len(result.Colors) == 0 || result.Colors[0].Ratio < 0.9 {
		t.Fatalf("colors = %#v", result.Colors)
	}
	var red, green, blue int
	if _, err := fmt.Sscanf(result.Colors[0].Hex, "#%02X%02X%02X", &red, &green, &blue); err != nil {
		t.Fatal(err)
	}
	if red < 240 || green > 10 || blue > 10 {
		t.Fatalf("opening color = %s", result.Colors[0].Hex)
	}
}

func TestAnalyzeVideoHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	_, err := assetmedia.AnalyzeVideo(ctx, bytes.NewReader([]byte("video")))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("canceled analysis took %s", time.Since(start))
	}
}
