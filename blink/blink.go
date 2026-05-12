package blink

import (
	"image"
	"image/draw"
	"math/rand"
)

// Frame is a single animation frame with its display duration.
type Frame struct {
	Image image.Image
	Delay int // 10ms units (1/100 sec), same as GIF and APNG centiseconds
}

// Preset defines blink timing parameters for a character personality.
type Preset struct {
	WaitMin           int     // minimum wait between blinks (10ms units)
	WaitMax           int     // maximum wait between blinks (10ms units)
	CloseTime         int     // closed-eye frame duration (10ms units)
	HalfTime          int     // half-open frame duration (10ms units)
	DoubleBlinkChance float64 // probability of a second blink following the first
	DoubleWaitMin     int     // minimum gap between double blinks (10ms units)
	DoubleWaitMax     int     // maximum gap between double blinks (10ms units)
}

var presets = map[string]Preset{
	// ふつう: natural human blink rhythm
	"normal": {
		WaitMin:           280,
		WaitMax:           480,
		CloseTime:         4,
		HalfTime:          7,
		DoubleBlinkChance: 0.25,
		DoubleWaitMin:     60,
		DoubleWaitMax:     100,
	},
	// のんびり: slow, drowsy blink
	"relaxed": {
		WaitMin:           450,
		WaitMax:           800,
		CloseTime:         5,
		HalfTime:          9,
		DoubleBlinkChance: 0.10,
		DoubleWaitMin:     80,
		DoubleWaitMax:     130,
	},
	// せわしない: rapid, fidgety blink
	"busy": {
		WaitMin:           100,
		WaitMax:           260,
		CloseTime:         3,
		HalfTime:          5,
		DoubleBlinkChance: 0.45,
		DoubleWaitMin:     40,
		DoubleWaitMax:     70,
	},
}

// GetPreset returns the named preset, falling back to "normal".
func GetPreset(name string) Preset {
	if p, ok := presets[name]; ok {
		return p
	}
	return presets["normal"]
}

func randBetween(min, max int) int {
	if min >= max {
		return min
	}
	return min + rand.Intn(max-min)
}

// GenerateFrames builds the full blink animation frame sequence.
//
// Closing is fast (normal → closed in one cut).
// Opening is gradual (closed → half → normal) when half is provided.
// half may be nil; the animation falls back to closed → normal.
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

	// addBlink appends one blink sequence: wait → close → (half →) normal.
	// waitDelay is the duration of the leading normal-face frame.
	addBlink := func(waitDelay int) {
		// Normal face (wait)
		frames = append(frames, Frame{Image: normal, Delay: waitDelay})
		// Fast close
		frames = append(frames, Frame{Image: closed, Delay: p.CloseTime})
		// Gradual open (half-eye if available)
		if hasHalf {
			frames = append(frames, Frame{Image: half, Delay: p.HalfTime})
		}
	}

	// Generate 3 blink cycles with randomised timing and occasional double blinks.
	for i := 0; i < 3; i++ {
		wait := randBetween(p.WaitMin, p.WaitMax)
		addBlink(wait)

		if rand.Float64() < p.DoubleBlinkChance {
			doubleWait := randBetween(p.DoubleWaitMin, p.DoubleWaitMax)
			addBlink(doubleWait)
		}
	}

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
