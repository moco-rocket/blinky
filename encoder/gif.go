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
// Each frame is composited onto opaque black before quantisation so that
// Floyd–Steinberg error diffusion operates only on fully-opaque pixels.
// Original transparency is restored afterward (GIF supports 1-bit alpha only).
func EncodeGIF(w io.Writer, frames []blink.Frame) error {
	if len(frames) == 0 {
		return nil
	}

	// 256-entry palette: transparent at index 0, Plan9 colours after.
	pal := make(color.Palette, 1, 256)
	pal[0] = color.RGBA{0, 0, 0, 0} // transparent sentinel
	for i, c := range palette.Plan9 {
		if i >= 255 {
			break
		}
		pal = append(pal, c)
	}

	anim := &gif.GIF{LoopCount: 0}

	// opaqueBlack is used as the composite background.
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
