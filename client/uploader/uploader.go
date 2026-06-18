package uploader

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	DefaultBlockSize = 256
	defaultFieldName = "file"
)

type Config struct {
	URL       string
	FieldName string
	BlockSize int
	Headers   map[string]string
	Timeout   time.Duration
}

type Client struct {
	cfg        Config
	httpClient *fasthttp.Client
}

type Response struct {
	StatusCode int
	Body       []byte
}

func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, errors.New("url is required")
	}
	if cfg.BlockSize <= 0 {
		cfg.BlockSize = DefaultBlockSize
	}
	if cfg.FieldName == "" {
		cfg.FieldName = defaultFieldName
	}

	return &Client{
		cfg:        cfg,
		httpClient: &fasthttp.Client{},
	}, nil
}

func (c *Client) UploadFile(path string) (*Response, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open source file: %w", err)
	}
	defer file.Close()

	return c.Upload(filepath.Base(path), file)
}

func (c *Client) Upload(filename string, src io.Reader) (*Response, error) {
	if filename == "" {
		filename = "stream"
	}
	if src == nil {
		return nil, errors.New("source reader is required")
	}

	body, contentType, waitBody := newMultipartStream(c.cfg.FieldName, filename, src, c.cfg.BlockSize)
	defer body.Close()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI(c.cfg.URL)
	req.Header.SetContentType(contentType)
	for key, value := range c.cfg.Headers {
		req.Header.Set(key, value)
	}
	req.SetBodyStream(body, -1)

	var err error
	if c.cfg.Timeout > 0 {
		err = c.httpClient.DoTimeout(req, resp, c.cfg.Timeout)
	} else {
		err = c.httpClient.Do(req, resp)
	}
	if err != nil {
		_ = body.CloseWithError(err)
		_ = waitBody()
		return nil, fmt.Errorf("send request: %w", err)
	}
	if err := waitBody(); err != nil {
		return nil, fmt.Errorf("write multipart body: %w", err)
	}

	result := &Response{
		StatusCode: resp.StatusCode(),
		Body:       append([]byte(nil), resp.Body()...),
	}
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return result, fmt.Errorf("unexpected response status: %d", result.StatusCode)
	}

	return result, nil
}

func newMultipartStream(fieldName, filename string, src io.Reader, blockSize int) (*io.PipeReader, string, func() error) {
	if blockSize <= 0 {
		blockSize = DefaultBlockSize
	}

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	errCh := make(chan error, 1)

	go func() {
		var err error
		defer close(errCh)

		part, err := mw.CreateFormFile(fieldName, filename)
		if err == nil {
			buf := make([]byte, blockSize)
			_, err = io.CopyBuffer(part, src, buf)
		}
		if closeErr := mw.Close(); err == nil {
			err = closeErr
		}

		if err != nil {
			_ = pw.CloseWithError(err)
			errCh <- err
			return
		}

		errCh <- pw.Close()
	}()

	return pr, mw.FormDataContentType(), func() error {
		return <-errCh
	}
}
