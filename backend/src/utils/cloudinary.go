package utils

import (
	"context"
	"errors"
	"mime/multipart"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

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
	cld, err := getCloudinaryClient()
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	resp, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID: filename,
		Folder:   "avatars",
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
