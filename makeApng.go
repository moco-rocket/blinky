package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/cia-rana/goapng"
)

func main() {
	inPaths := []string{
		"res/通常.png",
		"res/ジト目.png",
		"res/まばたき.png",
	}
	delaytime := []uint16{
		400,
		0,
		0,
		600,
		0,
		0,
		0,
		0,
	}
	frameIndex := []int{
		0,
		2,
		1,
		0,
		1,
		0,
		2,
		1,
	}
	outPath := "res/通常anim.png"

	// Assemble output image.
	outApng := &goapng.APNG{}
	i := 0
	for _, frameImage := range frameIndex {
		// Read image file.
		f, err := os.Open(inPaths[frameImage])
		if err != nil {
			fmt.Println(err)
			f.Close()
			return
		}
		inPng, err := png.Decode(f)
		if err != nil {
			fmt.Println(err)
			f.Close()
			return
		}
		f.Close()

		// Append a frame(type: *image.Image). First frame used as the default image.
		outApng.Images = append(outApng.Images, inPng)

		// Append a delay time(type: uint32) per frame in 10 milliseconds.
		// If it is 0, the decoder renders the next frame as quickly as possible.
		outApng.Delays = append(outApng.Delays, delaytime[i])
		i += 1
	}

	// Encode images to APNG image.
	f, err := os.Create(outPath)
	if err != nil {
		fmt.Println(err)
		f.Close()

		return
	}
	if err = goapng.EncodeAll(f, outApng); err != nil {
		fmt.Println(err)
		f.Close()
		os.Remove(outPath)
		return
	}
	f.Close()
}
