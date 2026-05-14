package encoder

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"sort"

	"blinky/blink"
)

// EncodeGIF writes an animated GIF to w.
//
// Each frame is composited onto opaque black before quantisation so that
// Floyd–Steinberg error diffusion operates only on fully-opaque pixels.
// Original transparency is restored afterward (GIF supports 1-bit alpha only).
func EncodeGIF(w io.Writer, frames []blink.Frame) error {
	if len(frames) == 0 {
		return nil
	}

	pal := buildPalette(frames)
	anim := &gif.GIF{LoopCount: 0}
	opaqueBlack := &image.Uniform{C: color.NRGBA{0, 0, 0, 255}}

	for _, f := range frames {
		b := f.Image.Bounds()

		// Composite onto opaque black so that transparent pixels have a
		// defined colour (black) rather than (0,0,0,0). This prevents
		// Floyd–Steinberg from spreading zero-alpha error into adjacent
		// opaque pixels at the transparent/opaque boundary.
		composited := image.NewNRGBA(b)
		draw.Draw(composited, b, opaqueBlack, image.Point{}, draw.Src)
		draw.Draw(composited, b, f.Image, b.Min, draw.Over)

		palImg := image.NewPaletted(b, pal)
		draw.FloydSteinberg.Draw(palImg, b, composited, b.Min)

		// Restore original transparency; GIF only supports 1-bit alpha.
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				_, _, _, a := f.Image.At(x, y).RGBA()
				if a < 0x8000 {
					palImg.SetColorIndex(x, y, 0)
				}
			}
		}

		anim.Image = append(anim.Image, palImg)
		anim.Delay = append(anim.Delay, f.Delay)
		anim.Disposal = append(anim.Disposal, gif.DisposalBackground)
	}

	return gif.EncodeAll(w, anim)
}

// buildPalette samples all opaque pixels from every frame and returns a
// 255-colour palette sorted by frequency (plus index 0 for transparency).
// For illustration-style images this preserves the exact colours of the
// artwork far better than a generic palette such as Plan9.
func buildPalette(frames []blink.Frame) color.Palette {
	counts := make(map[[3]byte]int)
	for _, f := range frames {
		b := f.Image.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, a := f.Image.At(x, y).RGBA()
				if a < 0x8000 {
					continue
				}
				counts[[3]byte{byte(r >> 8), byte(g >> 8), byte(bl >> 8)}]++
			}
		}
	}

	type entry struct {
		rgb [3]byte
		n   int
	}
	entries := make([]entry, 0, len(counts))
	for k, n := range counts {
		entries = append(entries, entry{k, n})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].n > entries[j].n })

	pal := make(color.Palette, 1, 256)
	pal[0] = color.RGBA{0, 0, 0, 0} // transparent sentinel
	limit := 255
	if len(entries) < limit {
		limit = len(entries)
	}
	for i := 0; i < limit; i++ {
		e := entries[i]
		pal = append(pal, color.RGBA{e.rgb[0], e.rgb[1], e.rgb[2], 255})
	}
	return pal
}
