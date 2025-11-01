package libs

import (
	"bytes"
	"context"
	"fmt"
	"imgop/src/helpers"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cshum/vipsgen/vips"
)

type ImageOptimizerHandler struct{}

func NewImageOptimizer() *ImageOptimizerHandler {
	return &ImageOptimizerHandler{}
}

func (imgop *ImageOptimizerHandler) Optimize(params helpers.ParamsOptimize) []byte {
	appEnv := helpers.GetAppEnv()
	// Validate if it is a proper url using simple reges
	imageUrl, err := url.Parse(params.Url)
	if err != nil {
		return []byte{}
	}

	// Get timeout from environment variable, default to 5 seconds
	timeout := time.Duration(appEnv.FETCH_TIMEOUT) * time.Second

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", imageUrl.String(), nil)
	if err != nil {
		return []byte{}
	}

	// Execute request with timeout
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return []byte{}
	}

	// Validate that the response is an image and get validated body reader
	validatedBody, err := validateImageFile(resp)
	if err != nil {
		return []byte{}
	}
	defer validatedBody.Close()

	// Create source from validated image body
	source := vips.NewSource(validatedBody)
	defer source.Close() // source needs to remain available during image lifetime

	image, err := vips.NewImageFromSource(source, &vips.LoadOptions{
		FailOnError: true, // Fail on first error
	})

	if err != nil {
		NewError(err)
		return []byte{}
	}

	originalWidth := image.Width()
	originalHeight := image.Height()

	var scale float64 = 1.0 // Default left as it is

	switch {
	case params.Width > 0 && params.Height == 0:
		// Only width is specified: scale proportionally based on width
		scale = float64(params.Width) / float64(originalWidth)
	case params.Height > 0 && params.Width == 0:
		// Only height is specified: scale proportionally based on height
		scale = float64(params.Height) / float64(originalHeight)
	case params.Width > 0 && params.Height > 0:
		// Both dimensions specified: calculate scale to fit within the box (contain)
		scaleW := float64(params.Width) / float64(originalWidth)
		scaleH := float64(params.Height) / float64(originalHeight)

		// We choose the smaller scale factor to ensure the image fits *inside* the box.
		scale = math.Min(scaleW, scaleH)
	}

	image.Resize(scale, nil)
	imageByte, err := image.WebpsaveBuffer(&vips.WebpsaveBufferOptions{
		Q:              params.Quality, // Quality factor (0-100)
		Effort:         4,              // Compression effort (0-6)
		SmartSubsample: true,           // Better chroma subsampling
	})

	if err != nil {
		NewError(err)
		return []byte{}
	}

	return imageByte
}

func NewError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// validateImageFile validates that the HTTP response contains a valid image file.
// It checks both Content-Type header and file signature (magic numbers).
// Returns a ReadCloser containing the validated image body, or an error if validation fails.
func validateImageFile(resp *http.Response) (io.ReadCloser, error) {
	// Validate Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !isImageContentType(contentType) {
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	// Read first bytes to verify image file signature (magic numbers)
	peekBuffer := make([]byte, 12)
	n, err := resp.Body.Read(peekBuffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	// Verify file signature matches known image formats
	if !isImageFileSignature(peekBuffer[:n]) {
		return nil, fmt.Errorf("invalid image file signature")
	}

	// Reconstruct the body with peeked bytes + remaining body
	bodyReader := io.MultiReader(bytes.NewReader(peekBuffer[:n]), resp.Body)

	// Return as ReadCloser (wrap reader with NopCloser to implement ReadCloser)
	return io.NopCloser(bodyReader), nil
}

// isImageContentType checks if the Content-Type header indicates an image
func isImageContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "" {
		return false
	}

	// Remove charset and other parameters (e.g., "image/jpeg; charset=utf-8")
	parts := strings.Split(contentType, ";")
	contentType = strings.TrimSpace(parts[0])

	// Check if it starts with "image/"
	return strings.HasPrefix(contentType, "image/")
}

// isImageFileSignature checks if the first bytes match known image file signatures (magic numbers)
func isImageFileSignature(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// JPEG: FF D8 FF
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}

	// PNG: 89 50 4E 47
	if len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}

	// GIF: 47 49 46 38 (GIF8)
	if len(data) >= 4 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return true
	}

	// WebP: Check for "RIFF" (4 bytes) followed by "WEBP" (at offset 8)
	if len(data) >= 12 &&
		data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 && // RIFF
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 { // WEBP
		return true
	}

	// BMP: 42 4D
	if len(data) >= 2 && data[0] == 0x42 && data[1] == 0x4D {
		return true
	}

	// TIFF: 49 49 2A 00 (little-endian) or 4D 4D 00 2A (big-endian)
	if len(data) >= 4 {
		if (data[0] == 0x49 && data[1] == 0x49 && data[2] == 0x2A && data[3] == 0x00) ||
			(data[0] == 0x4D && data[1] == 0x4D && data[2] == 0x00 && data[3] == 0x2A) {
			return true
		}
	}

	// HEIC/HEIF: Check for "ftyp" at offset 4
	if len(data) >= 8 && data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x70 {
		// Check for heic/heif brands
		if len(data) >= 12 {
			brand := string(data[8:12])
			if strings.Contains(brand, "heic") || strings.Contains(brand, "heif") || strings.Contains(brand, "mif1") {
				return true
			}
		}
	}

	return false
}
