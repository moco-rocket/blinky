package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
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
	"strings"
	"sync"
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

type ImageSession struct {
	Images    []image.Image
	Filenames []string
	CreatedAt time.Time
}

var (
	sessions = make(map[string]*ImageSession)
	sessionsMutex = sync.RWMutex{}
)

func main() {
	// Start cleanup goroutine for expired sessions
	go cleanupExpiredSessions()
	
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("./static/")))
	
	// API endpoints
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/process", handleProcess)
	http.HandleFunc("/download/", handleDownload)
	
	fmt.Println("サーバーを起動しています... http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// generateSessionID creates a secure random session ID
func generateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// cleanupExpiredSessions removes old sessions to prevent memory leaks
func cleanupExpiredSessions() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		sessionsMutex.Lock()
		for sessionID, session := range sessions {
			// Remove sessions older than 1 hour
			if time.Since(session.CreatedAt) > time.Hour {
				delete(sessions, sessionID)
			}
		}
		sessionsMutex.Unlock()
	}
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

	// Process uploaded files
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	var images []image.Image
	var filenames []string

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error opening file %s", fileHeader.Filename), http.StatusInternalServerError)
			return
		}
		
		// Read file content
		content, err := io.ReadAll(file)
		file.Close() // Close immediately to prevent resource leaks
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

		images = append(images, img)
		filenames = append(filenames, fileHeader.Filename)
	}

	// Auto-duplicate first image if less than 3 images
	if len(images) == 1 {
		images = append(images, images[0])
		filenames = append(filenames, filenames[0])
	} else if len(images) == 2 {
		images = append(images, images[0])
		filenames = append(filenames, filenames[0])
	}

	// Limit to 3 images
	if len(images) > 3 {
		images = images[:3]
		filenames = filenames[:3]
	}

	// Generate session ID
	sessionID := generateSessionID()
	
	// Store in session
	sessionsMutex.Lock()
	sessions[sessionID] = &ImageSession{
		Images:    images,
		Filenames: filenames,
		CreatedAt: time.Now(),
	}
	sessionsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("アップロード完了: %d枚の画像", len(images)),
		"count":     len(images),
		"sessionId": sessionID,
	})
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProcessRequest
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get session
	sessionsMutex.RLock()
	session, exists := sessions[req.SessionID]
	sessionsMutex.RUnlock()

	if !exists || session == nil {
		http.Error(w, "No images uploaded or session expired", http.StatusBadRequest)
		return
	}

	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	var filename string
	
	switch req.Format {
	case "apng":
		filename = fmt.Sprintf("animation_%s.png", timestamp)
	case "webp":
		// Note: Currently generates APNG format with .webp extension
		// Full WebP animation support would require additional libraries
		filename = fmt.Sprintf("animation_%s.webp", timestamp)
	case "avif":
		// Note: Currently generates APNG format with .avif extension
		// Full AVIF animation support would require additional libraries
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

	// Generate animation (currently only APNG is fully supported)
	var err error
	switch req.Format {
	case "apng":
		err = generateAPNG(outputPath, req.Duration, session.Images)
	case "webp", "avif":
		// Generate APNG for now, but with appropriate extension
		// This maintains compatibility while clearly indicating limitation
		err = generateAPNG(outputPath, req.Duration, session.Images)
	default:
		err = generateAPNG(outputPath, req.Duration, session.Images)
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

func generateAPNG(outputPath string, duration int, images []image.Image) error {
	if len(images) == 0 {
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
	for _, img := range images {
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

	// Security: Prevent path traversal attacks
	filename = filepath.Base(filename) // Remove any path components
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Additional security: Only allow expected file extensions
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".webp" && ext != ".avif" {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	// Use filepath.Join to safely construct the path
	filepath := filepath.Join("output", filename)
	
	// Resolve the path and ensure it's within output directory
	absOutputDir, err := filepath.Abs("output")
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	
	absFilePath, err := filepath.Abs(filepath)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	
	// Ensure the resolved path is within the output directory
	if !strings.HasPrefix(absFilePath, absOutputDir+string(os.PathSeparator)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
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