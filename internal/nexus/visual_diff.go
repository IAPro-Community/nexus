package nexus

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"

	// Deterministic decoding of the two capture formats Nexus produces.
	// Blank activation keeps image.Decode format-agnostic so JPEG captures
	// from the browser are comparable alongside PNG.
	_ "image/jpeg"
	"image/png"
)

// Visual diff (capability visual-diff).
//
// The comparison is deterministic: the same pair of screenshots with the
// same declared region and thresholds always produces the same result.
// Region and thresholds must be declared by the caller — the detector never
// invents them — and every result echoes its inputs so artifacts are
// traceable. The result never upgrades to "pass" implicitly: passing means
// the measured diff ratio is within the declared maximum.

var (
	// ErrScreenshotSizeMismatch reports images that cannot be compared
	// pixel-wise; callers must recapture at the same viewport.
	ErrScreenshotSizeMismatch = errors.New("visual diff: screenshot dimensions differ; recapture both sides at the same viewport")
	// ErrScreenshotUndecodable wraps format errors without leaking internals.
	ErrScreenshotUndecodable = errors.New("visual diff: screenshot is not a decodable PNG/JPEG image")
)

// VisualDiffRegion is the declared comparison region in pixels. Zero value
// means the whole image.
type VisualDiffRegion struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Contains reports whether the region is fully inside an image bounds.
func (r VisualDiffRegion) Contains(bounds image.Rectangle) bool {
	if r.X < 0 || r.Y < 0 || r.Width < 0 || r.Height < 0 {
		return false
	}
	if r.X+r.Width > bounds.Dx() || r.Y+r.Height > bounds.Dy() {
		return false
	}
	return true
}

// VisualDiffInput declares one comparison. PixelThreshold is the maximum
// per-channel delta treated as identical (0..255); MaxDiffRatio is the
// largest fraction of different pixels still considered passing (0..1).
type VisualDiffInput struct {
	Region         *VisualDiffRegion
	PixelThreshold uint8
	MaxDiffRatio   float64
}

// VisualDiffResult echoes the declared inputs and the measurement so every
// comparison artifact is traceable on its own.
type VisualDiffResult struct {
	TotalPixels      int               `json:"total_pixels"`
	ComparedPixels   int               `json:"compared_pixels"`
	DifferentPixels  int               `json:"different_pixels"`
	DiffRatio        float64           `json:"diff_ratio"`
	PixelThreshold   uint8             `json:"pixel_threshold"`
	MaxDiffRatio     float64           `json:"max_diff_ratio"`
	Region           *VisualDiffRegion `json:"region,omitempty"`
	Passed           bool              `json:"passed"`
	SizeMismatchNote string            `json:"size_mismatch_note,omitempty"`
}

// CompareScreenshots decodes both captures (PNG or JPEG), compares pixels
// inside the declared region, and returns the measurement. It is pure:
// identical inputs always yield an identical result.
func CompareScreenshots(before, after []byte, input VisualDiffInput) (VisualDiffResult, error) {
	beforeImg, err := decodeScreenshot(before)
	if err != nil {
		return VisualDiffResult{}, err
	}
	afterImg, err := decodeScreenshot(after)
	if err != nil {
		return VisualDiffResult{}, err
	}
	bounds := beforeImg.Bounds()
	afterBounds := afterImg.Bounds()
	if bounds.Dx() != afterBounds.Dx() || bounds.Dy() != afterBounds.Dy() {
		return VisualDiffResult{}, ErrScreenshotSizeMismatch
	}

	region := VisualDiffRegion{
		X:      bounds.Min.X,
		Y:      bounds.Min.Y,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}
	if input.Region != nil {
		region = *input.Region
		if !region.Contains(bounds) {
			return VisualDiffResult{}, fmt.Errorf(
				"visual diff: declared region %dx%d@%d,%d does not fit the capture %dx%d",
				region.Width, region.Height, region.X, region.Y, bounds.Dx(), bounds.Dy(),
			)
		}
	}

	result := VisualDiffResult{
		TotalPixels:    bounds.Dx() * bounds.Dy(),
		PixelThreshold: input.PixelThreshold,
		MaxDiffRatio:   input.MaxDiffRatio,
	}
	if input.Region != nil {
		regionCopy := *input.Region
		result.Region = &regionCopy
	}
	if input.MaxDiffRatio < 0 || input.MaxDiffRatio > 1 {
		return result, fmt.Errorf("visual diff: MaxDiffRatio must be within 0..1, got %v", input.MaxDiffRatio)
	}

	threshold := int(input.PixelThreshold)
	for y := region.Y; y < region.Y+region.Height; y++ {
		for x := region.X; x < region.X+region.Width; x++ {
			result.ComparedPixels++
			r1, g1, b1, _ := beforeImg.At(x, y).RGBA()
			r2, g2, b2, _ := afterImg.At(x, y).RGBA()
			if channelDelta(r1, r2) > threshold ||
				channelDelta(g1, g2) > threshold ||
				channelDelta(b1, b2) > threshold {
				result.DifferentPixels++
			}
		}
	}
	if result.ComparedPixels == 0 {
		return result, fmt.Errorf("visual diff: declared region has no pixels")
	}
	result.DiffRatio = float64(result.DifferentPixels) / float64(result.ComparedPixels)
	result.Passed = result.DiffRatio <= input.MaxDiffRatio
	return result, nil
}

// RenderDiffArtifact renders a traceable PNG artifact: grayscale before
// capture with differing pixels highlighted in red inside the declared
// region. Reviewers see exactly what the detector compared.
func RenderDiffArtifact(before, after []byte, input VisualDiffInput) ([]byte, VisualDiffResult, error) {
	beforeImg, err := decodeScreenshot(before)
	if err != nil {
		return nil, VisualDiffResult{}, err
	}
	afterImg, err := decodeScreenshot(after)
	if err != nil {
		return nil, VisualDiffResult{}, err
	}
	result, err := CompareScreenshots(before, after, input)
	if err != nil {
		return nil, result, err
	}

	bounds := beforeImg.Bounds()
	canvas := image.NewRGBA(bounds)
	// Grayscale base so the red highlights are unmistakable.
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := beforeImg.At(x, y).RGBA()
			lum := uint8((int(r)>>8*299 + int(g)>>8*587 + int(b)>>8*114) / 1000)
			canvas.Set(x, y, color.RGBA{R: lum, G: lum, B: lum, A: 255})
		}
	}

	region := VisualDiffRegion{X: bounds.Min.X, Y: bounds.Min.Y, Width: bounds.Dx(), Height: bounds.Dy()}
	if input.Region != nil {
		region = *input.Region
	}
	highlight := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	threshold := int(input.PixelThreshold)
	for y := region.Y; y < region.Y+region.Height; y++ {
		for x := region.X; x < region.X+region.Width; x++ {
			r1, g1, b1, _ := beforeImg.At(x, y).RGBA()
			r2, g2, b2, _ := afterImg.At(x, y).RGBA()
			if channelDelta(r1, r2) > threshold ||
				channelDelta(g1, g2) > threshold ||
				channelDelta(b1, b2) > threshold {
				canvas.Set(x, y, highlight)
			}
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, result, fmt.Errorf("visual diff: encoding artifact: %w", err)
	}
	return out.Bytes(), result, nil
}

func decodeScreenshot(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, ErrScreenshotUndecodable
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrScreenshotUndecodable
	}
	return img, nil
}

func channelDelta(a, b uint32) int {
	delta := int(a>>8) - int(b>>8)
	if delta < 0 {
		delta = -delta
	}
	return delta
}
