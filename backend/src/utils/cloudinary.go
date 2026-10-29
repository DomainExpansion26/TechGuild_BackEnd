package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gabriel-vasile/mimetype"
)

var ErrInvalidFileType = errors.New("invalid file type")

// getCloudinaryClient returns a configured Cloudinary client
func getCloudinaryClient() (*cloudinary.Cloudinary, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, errors.New("cloudinary credentials are not set in the environment")
	}

	return cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
}

// UploadPDFToCloudinary uploads a multipart file to Cloudinary as a raw file and returns the secure URL
func UploadPDFToCloudinary(file multipart.File, filename string) (string, error) {
	// Verify content is actually a PDF (not just a renamed file)
	mtype, err := detectMIME(file)
	if err != nil {
		return "", err
	}
	if mtype != "application/pdf" {
		return "", fmt.Errorf("%w: expected application/pdf, got %s", ErrInvalidFileType, mtype)
	}
	cld, err := getCloudinaryClient()
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	resp, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:     filename,
		Folder:       "resumes",
		ResourceType: "raw",
	})

	if err != nil {
		return "", err
	}

	return resp.SecureURL, nil
}

// UploadImageToCloudinary uploads an image file to Cloudinary and returns the secure URL
func UploadImageToCloudinary(file multipart.File, filename string) (string, error) {
	// Verify content is actually a supported image format
	mtype, err := detectMIME(file)
	if err != nil {
		return "", err
	}
	switch mtype {
	case "image/jpeg", "image/png", "image/webp":
		// ok
	default:
		return "", fmt.Errorf("%w: expected jpeg/png/webp, got %s", ErrInvalidFileType, mtype)
	}

	cld, err := getCloudinaryClient()
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	resp, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID: filename,
		Folder:   "avatars",
		Eager:    "w_256,h_256,c_fill|w_512,h_512,c_fill",
	})

	if err != nil {
		return "", err
	}

	return resp.SecureURL, nil
}

// DeleteFromCloudinary deletes a file from Cloudinary by its public ID
// resourceType should be "image" for avatars or "raw" for resumes/PDFs
func DeleteFromCloudinary(publicID string, resourceType string) error {
	cld, err := getCloudinaryClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	_, err = cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: resourceType,
	})

	return err
}

// UploadJSONToCloudinary uploads raw JSON bytes to Cloudinary as a raw file and returns the secure URL.
// Used for GDPR/DPDP data exports.
func UploadJSONToCloudinary(data []byte, filename string) (string, error) {
	cld, err := getCloudinaryClient()
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	reader := bytes.NewReader(data)

	resp, err := cld.Upload.Upload(ctx, reader, uploader.UploadParams{
		PublicID:     filename,
		Folder:       "exports",
		ResourceType: "raw",
	})
	if err != nil {
		return "", err
	}

	return resp.SecureURL, nil
}

// UploadVerificationDocument uploads verification documents
// (Aadhaar, PAN, GST, Selfie, etc.) to Cloudinary.
func UploadVerificationDocument(file multipart.File, filename string) (string, error) {

	cld, err := getCloudinaryClient()
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	resp, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID: filename,
		Folder:   "verification_documents",
	})

	if err != nil {
		return "", err
	}

	return resp.SecureURL, nil
}

// detectMIME inspects file content (not just the filename) and resets the
// file's read position so the caller can upload it afterwards.
func detectMIME(file multipart.File) (string, error) {
	mtype, err := mimetype.DetectReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to detect file type: %w", err)
	}
	// Rewind — detection consumed the initial bytes of the stream.
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to rewind file after detection: %w", err)
	}
	return mtype.String(), nil
}
