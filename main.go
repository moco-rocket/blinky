package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"blinky/blink"
	"blinky/encoder"

	_ "golang.org/x/image/webp"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("/process", handleProcess)

	log.Printf("Blinky listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// handleProcess is the sole API endpoint.
// It accepts a multipart form with image fields and returns the animation directly.
// No files are saved; no session state is kept.
func handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "リクエストの解析に失敗しました: "+err.Error(), http.StatusBadRequest)
		return
	}

	normal, err := readFormImage(r, "normal")
	if err != nil {
		http.Error(w, "通常顔の読み込みに失敗しました: "+err.Error(), http.StatusBadRequest)
		return
	}

	closed, err := readFormImage(r, "closed")
	if err != nil {
		http.Error(w, "閉じ目の読み込みに失敗しました: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Half-open eye is optional.
	var half image.Image
	if hf, _, hErr := r.FormFile("half"); hErr == nil {
		defer hf.Close()
		if data, rErr := io.ReadAll(hf); rErr == nil {
			if img, _, dErr := image.Decode(bytes.NewReader(data)); dErr == nil {
				half = img
			}
		}
	}

	preset := blink.GetPreset(r.FormValue("preset"))
	frames := blink.GenerateFrames(normal, closed, half, preset)

	// Encode into a buffer so we can set Content-Length and handle errors cleanly.
	var buf bytes.Buffer
	format := r.FormValue("format")

	switch format {
	case "gif":
		if err := encoder.EncodeGIF(&buf, frames); err != nil {
			http.Error(w, "GIF生成に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Content-Disposition", `attachment; filename="blink.gif"`)
	default: // apng
		if err := encoder.EncodeAPNG(&buf, frames); err != nil {
			http.Error(w, "APNG生成に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", `attachment; filename="blink.png"`)
	}

	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	_, _ = w.Write(buf.Bytes())
}

// maxImageDim is the maximum allowed width or height of an input image.
// This prevents memory exhaustion from very large images on Cloud Run instances
// with limited memory.
const maxImageDim = 4096

// readFormImage reads a named file field from a multipart form and decodes it
// as an image. Supports PNG, JPEG, and WebP (via registered decoders).
func readFormImage(r *http.Request, field string) (image.Image, error) {
	file, _, err := r.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("フィールド %q が見つかりません", field)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	b := img.Bounds()
	if b.Dx() > maxImageDim || b.Dy() > maxImageDim {
		return nil, fmt.Errorf("画像が大きすぎます（最大 %d px、実際 %dx%d）", maxImageDim, b.Dx(), b.Dy())
	}

	return img, nil
}
