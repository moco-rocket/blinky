package encoder

import (
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"io"

	"blinky/blink"
)

// EncodeGIF writes an animated GIF to w.
//
// Transparent pixels (alpha < 50%) are preserved using palette index 0.
// Opaque pixels are quantised to Plan9's 256-colour palette with
// Floyd–Steinberg dithering for the best quality within GIF's limits.
func EncodeGIF(w io.Writer, frames []blink.Frame) error {
	if len(frames) == 0 {
		return nil
	}

	// Build a 256-entry palette: transparent at index 0, Plan9 colours after.
	pal := make(color.Palette, 1, 256)
	pal[0] = color.RGBA{0, 0, 0, 0} // transparent sentinel
	for i, c := range palette.Plan9 {
		if i >= 255 {
			break
		}
		pal = append(pal, c)
	}

	anim := &gif.GIF{LoopCount: 0}

	for _, f := range frames {
		b := f.Image.Bounds()
		palImg := image.NewPaletted(b, pal)

		// Dither opaque pixels into the palette.
		draw.FloydSteinberg.Draw(palImg, b, f.Image, b.Min)

		// Override: any pixel with < 50% alpha maps to transparent (index 0).
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
