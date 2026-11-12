package nexus

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func encodeTestPNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func solidImage(t *testing.T, width, height int, c color.RGBA) image.Image {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestCompareScreenshotsIdenticalPasses(t *testing.T) {
	before := encodeTestPNG(t, solidImage(t, 8, 8, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	after := encodeTestPNG(t, solidImage(t, 8, 8, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	result, err := CompareScreenshots(before, after, VisualDiffInput{PixelThreshold: 4, MaxDiffRatio: 0.01})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.DifferentPixels != 0 || result.DiffRatio != 0 {
		t.Fatalf("identical captures must pass: %+v", result)
	}
	if result.ComparedPixels != 64 || result.TotalPixels != 64 {
		t.Fatalf("region defaults to full capture: %+v", result)
	}
}

func TestCompareScreenshotsDetectsChangedPixels(t *testing.T) {
	before := encodeTestPNG(t, solidImage(t, 8, 8, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	changed := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			c := color.RGBA{R: 10, G: 20, B: 30, A: 255}
			if x >= 6 {
				c = color.RGBA{R: 250, G: 20, B: 30, A: 255}
			}
			changed.SetRGBA(x, y, c)
		}
	}
	after := encodeTestPNG(t, changed)
	result, err := CompareScreenshots(before, after, VisualDiffInput{PixelThreshold: 4, MaxDiffRatio: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	if result.DifferentPixels != 16 {
		t.Fatalf("different pixels = %d, want 16 (columns 6,7)", result.DifferentPixels)
	}
	if result.Passed {
		t.Fatalf("ratio %v exceeds declared maximum 0.1", result.DiffRatio)
	}
}

func TestCompareScreenshotsThresholdTolerance(t *testing.T) {
	before := encodeTestPNG(t, solidImage(t, 4, 4, color.RGBA{R: 100, G: 100, B: 100, A: 255}))
	after := encodeTestPNG(t, solidImage(t, 4, 4, color.RGBA{R: 110, G: 100, B: 100, A: 255}))
	within, err := CompareScreenshots(before, after, VisualDiffInput{PixelThreshold: 10, MaxDiffRatio: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !within.Passed {
		t.Fatalf("delta 10 at threshold 10 must pass: %+v", within)
	}
	beyond, err := CompareScreenshots(before, after, VisualDiffInput{PixelThreshold: 9, MaxDiffRatio: 0})
	if err != nil {
		t.Fatal(err)
	}
	if beyond.Passed || beyond.DifferentPixels != 16 {
		t.Fatalf("delta 10 at threshold 9 must fail: %+v", beyond)
	}
}

func TestCompareScreenshotsRegionScopesComparison(t *testing.T) {
	before := encodeTestPNG(t, solidImage(t, 8, 8, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	changed := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			c := color.RGBA{R: 10, G: 20, B: 30, A: 255}
			if y >= 6 {
				c = color.RGBA{R: 250, G: 20, B: 30, A: 255}
			}
			changed.SetRGBA(x, y, c)
		}
	}
	after := encodeTestPNG(t, changed)
	region := VisualDiffRegion{X: 0, Y: 0, Width: 8, Height: 4}
	result, err := CompareScreenshots(before, after, VisualDiffInput{
		Region: &region, PixelThreshold: 4, MaxDiffRatio: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DifferentPixels != 0 || !result.Passed {
		t.Fatalf("changes outside declared region must be ignored: %+v", result)
	}
	if result.Region == nil || result.Region.Height != 4 {
		t.Fatalf("result must echo declared region: %+v", result.Region)
	}
}

func TestCompareScreenshotsDeterministic(t *testing.T) {
	before := encodeTestPNG(t, solidImage(t, 6, 6, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
	after := encodeTestPNG(t, solidImage(t, 6, 6, color.RGBA{R: 200, G: 0, B: 0, A: 255}))
	input := VisualDiffInput{PixelThreshold: 4, MaxDiffRatio: 0.5}
	first, err := CompareScreenshots(before, after, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompareScreenshots(before, after, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.DifferentPixels != second.DifferentPixels || first.DiffRatio != second.DiffRatio {
		t.Fatalf("comparison must be deterministic: %+v vs %+v", first, second)
	}
}

func TestCompareScreenshotsRejectsSizeMismatchAndInvalidInputs(t *testing.T) {
	before := encodeTestPNG(t, solidImage(t, 8, 8, color.RGBA{R: 1, G: 2, B: 3, A: 255}))
	after := encodeTestPNG(t, solidImage(t, 4, 4, color.RGBA{R: 1, G: 2, B: 3, A: 255}))
	if _, err := CompareScreenshots(before, after, VisualDiffInput{MaxDiffRatio: 0.1}); err != ErrScreenshotSizeMismatch {
		t.Fatalf("size mismatch must be explicit, got %v", err)
	}
	if _, err := CompareScreenshots([]byte("not-an-image"), after, VisualDiffInput{MaxDiffRatio: 0.1}); err != ErrScreenshotUndecodable {
		t.Fatalf("undecodable input must be explicit, got %v", err)
	}
	outside := VisualDiffRegion{X: 4, Y: 0, Width: 8, Height: 8}
	if _, err := CompareScreenshots(before, before, VisualDiffInput{Region: &outside, MaxDiffRatio: 0.1}); err == nil {
		t.Fatal("region outside capture must be rejected")
	}
}

func TestRenderDiffArtifactHighlightsDifferences(t *testing.T) {
	before := encodeTestPNG(t, solidImage(t, 6, 6, color.RGBA{R: 40, G: 40, B: 40, A: 255}))
	after := encodeTestPNG(t, solidImage(t, 6, 6, color.RGBA{R: 240, G: 40, B: 40, A: 255}))
	artifact, result, err := RenderDiffArtifact(before, after, VisualDiffInput{PixelThreshold: 4, MaxDiffRatio: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.DifferentPixels != 36 {
		t.Fatalf("all pixels differ: %+v", result)
	}
	decoded, err := png.Decode(bytes.NewReader(artifact))
	if err != nil {
		t.Fatalf("artifact must be a valid PNG: %v", err)
	}
	highlights := 0
	b := decoded.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := decoded.At(x, y).RGBA()
			if int(r>>8) == 255 && int(g>>8) == 0 && int(bl>>8) == 0 {
				highlights++
			}
		}
	}
	if highlights != 36 {
		t.Fatalf("artifact must highlight every differing pixel, got %d", highlights)
	}
}
