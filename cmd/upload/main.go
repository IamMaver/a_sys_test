package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"a_sys_test/client/uploader"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:8080/upload", "upload endpoint")
	file := flag.String("file", "", "source file path")
	field := flag.String("field", "file", "multipart field name")
	blockSize := flag.Int("block-size", uploader.DefaultBlockSize, "read buffer size in bytes")
	timeout := flag.Duration("timeout", 30*time.Second, "request timeout")
	flag.Parse()

	if *file == "" {
		log.Fatal("file is required")
	}

	client, err := uploader.New(uploader.Config{
		URL:       *url,
		FieldName: *field,
		BlockSize: *blockSize,
		Timeout:   *timeout,
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.UploadFile(*file)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("status=%d body=%s\n", resp.StatusCode, resp.Body)
}
