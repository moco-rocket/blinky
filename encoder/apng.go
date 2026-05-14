// Package encoder provides APNG and GIF animation encoders.
package encoder

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/draw"
	"io"

	"blinky/blink"
)

var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// EncodeAPNG writes an Animated PNG to w.
//
// Frame delays are interpreted as centiseconds (1/100 sec), matching GIF
// and the goapng convention. The first frame is also used as the PNG default
// image so non-APNG viewers show it as a still image.
func EncodeAPNG(w io.Writer, frames []blink.Frame) error {
	if len(frames) == 0 {
		return nil
	}

	// Encode every frame to raw PNG bytes so we can extract IDAT chunks.
	// encodeFrameRGBA always produces colorType=6 (RGBA) regardless of
	// the source image type (JPEG→YCbCr, PNG→NRGBA, WebP→RGBA, etc.).
	// Without this, JPEG sources produce RGB PNGs (colorType=2, 3 bytes/px)
	// while PNG sources with transparency produce RGBA PNGs (colorType=6,
	// 4 bytes/px). The APNG global IHDR from frame 0 would then mismatch
	// the subsequent frames' byte layout, causing colour corruption on every
	// blink frame.
	pngData := make([][]byte, len(frames))
	for i, f := range frames {
		data, err := encodeFrameRGBA(f.Image)
		if err != nil {
			return err
		}
		pngData[i] = data
	}

	// Extract IHDR from first frame.
	firstChunks, err := parsePNGChunks(pngData[0])
	if err != nil {
		return err
	}

	// ── Write APNG ────────────────────────────────────────────────────────────

	if _, err := w.Write(pngSignature); err != nil {
		return err
	}

	// IHDR
	for _, c := range firstChunks {
		if c.typ == "IHDR" {
			if err := writeChunk(w, "IHDR", c.data); err != nil {
				return err
			}
			break
		}
	}

	// acTL (animation control): num_frames, num_plays=0 (infinite)
	actl := make([]byte, 8)
	binary.BigEndian.PutUint32(actl[0:4], uint32(len(frames)))
	binary.BigEndian.PutUint32(actl[4:8], 0)
	if err := writeChunk(w, "acTL", actl); err != nil {
		return err
	}

	var seq uint32

	for i, f := range frames {
		// fcTL (frame control)
		fctl := make([]byte, 26)
		binary.BigEndian.PutUint32(fctl[0:4], seq)
		seq++

		bounds := f.Image.Bounds()
		binary.BigEndian.PutUint32(fctl[4:8], uint32(bounds.Dx()))
		binary.BigEndian.PutUint32(fctl[8:12], uint32(bounds.Dy()))
		binary.BigEndian.PutUint32(fctl[12:16], 0) // x_offset
		binary.BigEndian.PutUint32(fctl[16:20], 0) // y_offset
		// delay_num / delay_den: f.Delay centiseconds → delay_num=f.Delay, den=100
		binary.BigEndian.PutUint16(fctl[20:22], uint16(f.Delay))
		binary.BigEndian.PutUint16(fctl[22:24], 100)
		fctl[24] = 0 // dispose_op: APNG_DISPOSE_OP_NONE
		fctl[25] = 1 // blend_op:   APNG_BLEND_OP_OVER

		if err := writeChunk(w, "fcTL", fctl); err != nil {
			return err
		}

		// Extract IDAT chunks from this frame's PNG.
		chunks, err := parsePNGChunks(pngData[i])
		if err != nil {
			return err
		}

		for _, c := range chunks {
			if c.typ != "IDAT" {
				continue
			}
			if i == 0 {
				// First frame: write as IDAT (also serves as the default PNG image).
				if err := writeChunk(w, "IDAT", c.data); err != nil {
					return err
				}
			} else {
				// Subsequent frames: wrap in fdAT with a sequence number prefix.
				fdat := make([]byte, 4+len(c.data))
				binary.BigEndian.PutUint32(fdat[0:4], seq)
				seq++
				copy(fdat[4:], c.data)
				if err := writeChunk(w, "fdAT", fdat); err != nil {
					return err
				}
			}
		}
	}

	// IEND
	return writeChunk(w, "IEND", nil)
}

// encodeFrameRGBA encodes img as a minimal PNG with colorType=6 (RGBA), always.
// Go's standard png.Encode chooses colorType=2 (RGB) when all pixels are fully
// opaque — which is the case for JPEG-decoded images. Mixing colorType=2 and
// colorType=6 frames in one APNG causes the browser to misread the RGBA byte
// stream as RGB, corrupting every pixel in the affected frames.
func encodeFrameRGBA(img image.Image) ([]byte, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// Always work with a freshly drawn NRGBA so pixels are R,G,B,A bytes.
	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(nrgba, nrgba.Bounds(), img, b.Min, draw.Src)

	// Build unfiltered scanlines: [filter_byte=0] [RGBA pixels…] per row.
	raw := make([]byte, h*(1+w*4))
	for y := 0; y < h; y++ {
		row := raw[y*(1+w*4):]
		// row[0] = 0 already (filter None)
		copy(row[1:], nrgba.Pix[y*nrgba.Stride:y*nrgba.Stride+w*4])
	}

	// zlib-compress the scanlines.
	var cBuf bytes.Buffer
	zw, err := zlib.NewWriterLevel(&cBuf, zlib.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err = zw.Write(raw); err != nil {
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}

	// Assemble a minimal valid PNG file.
	var buf bytes.Buffer
	buf.Write(pngSignature)

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(h))
	ihdr[8] = 8 // bit depth = 8
	ihdr[9] = 6 // color type = RGBA (bytes 10–12 stay 0)
	if err = writeChunk(&buf, "IHDR", ihdr); err != nil {
		return nil, err
	}
	if err = writeChunk(&buf, "IDAT", cBuf.Bytes()); err != nil {
		return nil, err
	}
	if err = writeChunk(&buf, "IEND", nil); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// pngChunk holds the type and data of a PNG chunk.
type pngChunk struct {
	typ  string
	data []byte
}

// parsePNGChunks splits raw PNG bytes (after the 8-byte signature) into chunks.
func parsePNGChunks(data []byte) ([]pngChunk, error) {
	if len(data) < 8 {
		return nil, io.ErrUnexpectedEOF
	}
	data = data[8:] // skip signature

	var chunks []pngChunk
	for len(data) >= 12 {
		length := binary.BigEndian.Uint32(data[0:4])
		if int(length)+12 > len(data) {
			break
		}
		typ := string(data[4:8])
		payload := make([]byte, length)
		copy(payload, data[8:8+length])
		chunks = append(chunks, pngChunk{typ, payload})
		data = data[8+length+4:] // advance past length+type+data+crc
	}
	return chunks, nil
}

// writeChunk writes a single PNG/APNG chunk to w.
func writeChunk(w io.Writer, typ string, data []byte) error {
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}

	h := crc32.NewIEEE()
	h.Write([]byte(typ))
	h.Write(data)

	if _, err := w.Write([]byte(typ)); err != nil {
		return err
	}
	if len(data) > 0 {
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	var crc [4]byte
	binary.BigEndian.PutUint32(crc[:], h.Sum32())
	_, err := w.Write(crc[:])
	return err
}
