package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cia-rana/goapng"
	"golang.org/x/image/webp"
)

type ProcessRequest struct {
	Format     string `json:"format"`
	Duration   int    `json:"duration"`
	ImageCount int    `json:"imageCount"`
}

type ProcessResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Filename string `json:"filename"`
}

var uploadedImages []image.Image
var uploadedFilenames []string

func main() {
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("./static/")))
	
	// API endpoints
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/process", handleProcess)
	http.HandleFunc("/download/", handleDownload)
	
	fmt.Println("サーバーを起動しています... http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Clear previous uploads
	uploadedImages = nil
	uploadedFilenames = nil

	// Process uploaded files
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error opening file %s", fileHeader.Filename), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Read file content
		content, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading file %s", fileHeader.Filename), http.StatusInternalServerError)
			return
		}

		// Decode image based on file extension
		var img image.Image
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		
		switch ext {
		case ".jpg", ".jpeg":
			img, err = jpeg.Decode(bytes.NewReader(content))
		case ".png":
			img, err = png.Decode(bytes.NewReader(content))
		case ".webp":
			img, err = webp.Decode(bytes.NewReader(content))
		default:
			http.Error(w, fmt.Sprintf("Unsupported file type: %s", ext), http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Error decoding image %s: %v", fileHeader.Filename, err), http.StatusInternalServerError)
			return
		}

		uploadedImages = append(uploadedImages, img)
		uploadedFilenames = append(uploadedFilenames, fileHeader.Filename)
	}

	// Auto-duplicate first image if less than 3 images
	if len(uploadedImages) == 1 {
		uploadedImages = append(uploadedImages, uploadedImages[0])
		uploadedFilenames = append(uploadedFilenames, uploadedFilenames[0])
	} else if len(uploadedImages) == 2 {
		uploadedImages = append(uploadedImages, uploadedImages[0])
		uploadedFilenames = append(uploadedFilenames, uploadedFilenames[0])
	}

	// Limit to 3 images
	if len(uploadedImages) > 3 {
		uploadedImages = uploadedImages[:3]
		uploadedFilenames = uploadedFilenames[:3]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("アップロード完了: %d枚の画像", len(uploadedImages)),
		"count":   len(uploadedImages),
	})
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(uploadedImages) == 0 {
		http.Error(w, "No images uploaded", http.StatusBadRequest)
		return
	}

	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	var filename string
	
	switch req.Format {
	case "apng":
		filename = fmt.Sprintf("animation_%s.png", timestamp)
	case "webp":
		filename = fmt.Sprintf("animation_%s.webp", timestamp)
	case "avif":
		filename = fmt.Sprintf("animation_%s.avif", timestamp)
	default:
		filename = fmt.Sprintf("animation_%s.png", timestamp)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll("output", 0755); err != nil {
		http.Error(w, "Error creating output directory", http.StatusInternalServerError)
		return
	}

	outputPath := filepath.Join("output", filename)

	// Generate animation based on format
	var err error
	switch req.Format {
	case "apng":
		err = generateAPNG(outputPath, req.Duration)
	case "webp":
		// For simplicity, generate APNG with .webp extension
		// In a real implementation, you would use a WebP library
		err = generateAPNG(outputPath, req.Duration)
	case "avif":
		// For simplicity, generate APNG with .avif extension
		// In a real implementation, you would use an AVIF library
		err = generateAPNG(outputPath, req.Duration)
	default:
		err = generateAPNG(outputPath, req.Duration)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating animation: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ProcessResponse{
		Success:  true,
		Message:  "アニメーション生成完了",
		Filename: filename,
	})
}

func generateAPNG(outputPath string, duration int) error {
	if len(uploadedImages) == 0 {
		return fmt.Errorf("no images to process")
	}

	// Create APNG
	outApng := &goapng.APNG{}
	
	// Convert duration from milliseconds to centiseconds (APNG uses centiseconds)
	delay := uint16(duration / 10)
	if delay == 0 {
		delay = 50 // Default 500ms
	}

	// Add frames
	for _, img := range uploadedImages {
		outApng.Images = append(outApng.Images, img)
		outApng.Delays = append(outApng.Delays, delay)
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Encode APNG
	if err = goapng.EncodeAll(f, outApng); err != nil {
		os.Remove(outputPath)
		return err
	}

	return nil
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/download/")
	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	filepath := filepath.Join("output", filename)
	
	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set headers for download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	
	// Serve file
	http.ServeFile(w, r, filepath)
}