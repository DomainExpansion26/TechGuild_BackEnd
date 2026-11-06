package utils

import (
	"errors"
	"testing"
)

func TestValidateFile_AllowedExtensions(t *testing.T) {
	allowedFiles := []string{
		"contract.pdf",
		"architecture.xml",
		"proposal.doc",
		"documentation.docx",
		"README.md",
		"SPEC.MD",
	}

	for _, fname := range allowedFiles {
		err := ValidateFile(fname, 1024, 10*1024*1024)
		if err != nil {
			t.Errorf("expected file %s to be valid, got: %v", fname, err)
		}
	}
}

func TestValidateFile_DisallowedExtensions(t *testing.T) {
	disallowedFiles := []string{
		"malware.exe",
		"script.sh",
		"data.zip",
		"photo.jpg",
		"audio.mp3",
		"video.mp4",
	}

	for _, fname := range disallowedFiles {
		err := ValidateFile(fname, 1024, 10*1024*1024)
		if err == nil {
			t.Errorf("expected file %s to be rejected", fname)
		}
		if !errors.Is(err, ErrFileTypeNotAllowed) {
			t.Errorf("expected ErrFileTypeNotAllowed for %s, got: %v", fname, err)
		}
	}
}

func TestValidateFile_SizeExceeded(t *testing.T) {
	err := ValidateFile("doc.pdf", 20*1024*1024, 10*1024*1024)
	if err == nil {
		t.Fatal("expected error when file size exceeds limit")
	}
	if !errors.Is(err, ErrFileSizeExceeded) {
		t.Fatalf("expected ErrFileSizeExceeded, got: %v", err)
	}
}

func TestValidateFile_EmptyFile(t *testing.T) {
	err := ValidateFile("doc.pdf", 0, 10*1024*1024)
	if err == nil {
		t.Fatal("expected error for 0-byte file")
	}
	if !errors.Is(err, ErrEmptyFile) {
		t.Fatalf("expected ErrEmptyFile, got: %v", err)
	}
}

func TestDefaultVirusScanner(t *testing.T) {
	scanner := &DefaultVirusScanner{}
	clean, details, err := scanner.ScanFile("test.pdf", []byte("%PDF-1.4..."))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !clean {
		t.Errorf("expected clean file, got details: %s", details)
	}
}

