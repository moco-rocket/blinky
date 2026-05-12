package blink

import (
	"image"
	"image/draw"
)

// Frame is a single animation frame with its display duration.
type Frame struct {
	Image image.Image
	Delay int // 10ms units (1/100 sec), same as GIF and APNG centiseconds
}

// Preset defines blink timing for one character personality.
// The animation is always: ぱち（単発） → wait → ぱちぱち（連続）
type Preset struct {
	WaitSingle int // wait before the single blink (10ms units)
	WaitDouble int // wait before the double blink (10ms units)
	DoubleWait int // gap between the two blinks in a double blink (10ms units)
	CloseTime  int // closed-eye frame duration (10ms units)
	HalfTime   int // half-open frame duration (10ms units)
}

var presets = map[string]Preset{
	// ふつう: natural human blink rhythm
	"normal": {
		WaitSingle: 350,
		WaitDouble: 400,
		DoubleWait: 80,
		CloseTime:  4,
		HalfTime:   7,
	},
	// のんびり: slow, drowsy blink
	"relaxed": {
		WaitSingle: 600,
		WaitDouble: 700,
		DoubleWait: 110,
		CloseTime:  5,
		HalfTime:   9,
	},
	// せわしない: rapid, fidgety blink
	"busy": {
		WaitSingle: 160,
		WaitDouble: 200,
		DoubleWait: 50,
		CloseTime:  3,
		HalfTime:   5,
	},
}

// GetPreset returns the named preset, falling back to "normal".
func GetPreset(name string) Preset {
	if p, ok := presets[name]; ok {
		return p
	}
	return presets["normal"]
}

// GenerateFrames builds a fixed blink sequence: ぱち → wait → ぱちぱち.
//
// Closing is fast (normal → closed in one frame).
// Opening is gradual (closed → half → normal) when half is provided.
// If half is nil the animation falls back to closed → normal.
func GenerateFrames(normal, closed, half image.Image, p Preset) []Frame {
	hasHalf := half != nil

	toNorm := []image.Image{normal, closed}
	if hasHalf {
		toNorm = append(toNorm, half)
	}
	normalized := normalizeImages(toNorm)

	normal = normalized[0]
	closed = normalized[1]
	if hasHalf {
		half = normalized[2]
	}

	var frames []Frame

	// blink appends: [normal wait] [closed] [half?]
	blink := func(wait int) {
		frames = append(frames, Frame{Image: normal, Delay: wait})
		frames = append(frames, Frame{Image: closed, Delay: p.CloseTime})
		if hasHalf {
			frames = append(frames, Frame{Image: half, Delay: p.HalfTime})
		}
	}

	// ぱち（単発瞬き）
	blink(p.WaitSingle)

	// ぱちぱち（二連瞬き）: first blink → short normal → second blink
	blink(p.WaitDouble)
	blink(p.DoubleWait)

	return frames
}

// normalizeImages ensures all images share the same canvas size.
// Smaller images are centred on a transparent canvas.
func normalizeImages(images []image.Image) []image.Image {
	if len(images) == 0 {
		return images
	}

	maxW, maxH := 0, 0
	for _, img := range images {
		b := img.Bounds()
		if b.Dx() > maxW {
			maxW = b.Dx()
		}
		if b.Dy() > maxH {
			maxH = b.Dy()
		}
	}

	result := make([]image.Image, len(images))
	for i, img := range images {
		b := img.Bounds()
		if b.Dx() == maxW && b.Dy() == maxH {
			result[i] = img
			continue
		}
		// NewNRGBA zero-initialises to transparent.
		canvas := image.NewNRGBA(image.Rect(0, 0, maxW, maxH))
		xOff := (maxW - b.Dx()) / 2
		yOff := (maxH - b.Dy()) / 2
		draw.Draw(canvas, image.Rect(xOff, yOff, xOff+b.Dx(), yOff+b.Dy()), img, b.Min, draw.Over)
		result[i] = canvas
	}

	return result
}
