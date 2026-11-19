package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	ErrFileTypeNotAllowed = errors.New("file type not allowed: must be one of .pdf, .xml, .doc, .docx, .md")
	ErrFileSizeExceeded   = errors.New("file size exceeds maximum allowed limit")
	ErrEmptyFile          = errors.New("file cannot be empty")
)

// AllowedFileExtensions contains the allowed extensions specified in the specification (pdf, xml, docs, .md)
var AllowedFileExtensions = map[string]bool{
	".pdf":  true,
	".xml":  true,
	".doc":  true,
	".docx": true,
	".md":   true,
}

// AllowedMIMETypes contains allowed MIME types for validation
var AllowedMIMETypes = map[string]bool{
	"application/pdf": true,
	"application/xml": true,
	"text/xml":        true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"text/markdown":        true,
	"text/plain":           true,
	"application/octet-stream": true, // fallback
}

// ValidateFile checks whether the given filename and size conform to requirements
func ValidateFile(filename string, sizeBytes int64, maxSizeBytes int64) error {
	if sizeBytes <= 0 {
		return ErrEmptyFile
	}

	if maxSizeBytes > 0 && sizeBytes > maxSizeBytes {
		return fmt.Errorf("%w: max %d bytes, got %d bytes", ErrFileSizeExceeded, maxSizeBytes, sizeBytes)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !AllowedFileExtensions[ext] {
		return fmt.Errorf("%w: extension '%s'", ErrFileTypeNotAllowed, ext)
	}

	return nil
}

// VirusScanHook is a pluggable virus scanning interface as noted in open items
type VirusScanHook interface {
	ScanFile(filename string, content []byte) (isClean bool, threatDetails string, err error)
}

// DefaultVirusScanner is the baseline scanner implementation (vendor integration open item)
type DefaultVirusScanner struct{}

func (s *DefaultVirusScanner) ScanFile(filename string, content []byte) (bool, string, error) {
	// Baseline clean check - can be wired to ClamAV, Cloudflare scanner, or third-party vendor
	return true, "", nil
}

