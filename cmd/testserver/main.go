package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var uploadDir string

func main() {
	flag.StringVar(&uploadDir, "upload-dir", "uploads", "directory for received files")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", handleUpload)

	addr := "127.0.0.1:8080"
	log.Printf("listening on http://%s/upload", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var total int64
	var saved []string
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if part.FileName() == "" {
			n, err := io.Copy(io.Discard, part)
			_ = part.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			total += n
			log.Printf("received form field=%q bytes=%d", part.FormName(), n)
			continue
		}

		path, file, err := createUploadFile()
		if err != nil {
			_ = part.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		n, copyErr := io.Copy(file, part)
		closeFileErr := file.Close()
		_ = part.Close()
		if copyErr != nil {
			http.Error(w, copyErr.Error(), http.StatusBadRequest)
			return
		}
		if closeFileErr != nil {
			http.Error(w, closeFileErr.Error(), http.StatusInternalServerError)
			return
		}

		total += n
		saved = append(saved, path)
		log.Printf("saved part field=%q filename=%q path=%q bytes=%d", part.FormName(), part.FileName(), path, n)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "received=%d\nsaved=%v\n", total, saved)
}

func createUploadFile() (string, *os.File, error) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", nil, fmt.Errorf("create upload dir: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405_000000000")
	path := filepath.Join(uploadDir, "upload_"+timestamp+".bin")

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return "", nil, fmt.Errorf("create upload file: %w", err)
	}

	return path, file, nil
}
