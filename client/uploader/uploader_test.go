package uploader

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func TestNewMultipartStreamBuildsReadableMultipartBody(t *testing.T) {
	const payload = "hello streaming multipart"

	body, contentType, waitBody := newMultipartStream("upload", "sample.txt", strings.NewReader(payload), 4)
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if err := waitBody(); err != nil {
		t.Fatalf("wait stream writer: %v", err)
	}

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parse content type: %v", err)
	}

	mr := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
	part, err := mr.NextPart()
	if err != nil {
		t.Fatalf("next part: %v", err)
	}
	if got := part.FormName(); got != "upload" {
		t.Fatalf("form name = %q, want %q", got, "upload")
	}
	if got := part.FileName(); got != "sample.txt" {
		t.Fatalf("file name = %q, want %q", got, "sample.txt")
	}

	got, err := io.ReadAll(part)
	if err != nil {
		t.Fatalf("read part: %v", err)
	}
	if string(got) != payload {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	client, err := New(Config{URL: "http://127.0.0.1:8080/upload"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if client.cfg.BlockSize != DefaultBlockSize {
		t.Fatalf("block size = %d, want %d", client.cfg.BlockSize, DefaultBlockSize)
	}
	if client.cfg.FieldName != defaultFieldName {
		t.Fatalf("field name = %q, want %q", client.cfg.FieldName, defaultFieldName)
	}
}
