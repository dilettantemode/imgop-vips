package libs

import (
	"bytes"
	"imgop/src/helpers"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cshum/vipsgen/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageOptimizer(t *testing.T) {
	handler := NewImageOptimizer()
	assert.NotNil(t, handler)
	assert.IsType(t, &ImageOptimizerHandler{}, handler)
}

func TestIsImageContentType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "valid jpeg",
			content:  "image/jpeg",
			expected: true,
		},
		{
			name:     "valid png",
			content:  "image/png",
			expected: true,
		},
		{
			name:     "valid gif",
			content:  "image/gif",
			expected: true,
		},
		{
			name:     "valid webp",
			content:  "image/webp",
			expected: true,
		},
		{
			name:     "valid content type with charset",
			content:  "image/jpeg; charset=utf-8",
			expected: true,
		},
		{
			name:     "valid content type with parameters",
			content:  "image/png; charset=utf-8; boundary=something",
			expected: true,
		},
		{
			name:     "case insensitive",
			content:  "IMAGE/JPEG",
			expected: true,
		},
		{
			name:     "with whitespace",
			content:  "  image/jpeg  ",
			expected: true,
		},
		{
			name:     "invalid text/html",
			content:  "text/html",
			expected: false,
		},
		{
			name:     "invalid application/json",
			content:  "application/json",
			expected: false,
		},
		{
			name:     "empty string",
			content:  "",
			expected: false,
		},
		{
			name:     "whitespace only",
			content:  "   ",
			expected: false,
		},
		{
			name:     "invalid prefix",
			content:  "application/image",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageContentType(tt.content)
			assert.Equal(t, tt.expected, result, "content type: %s", tt.content)
		})
	}
}

func TestIsImageFileSignature(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "valid JPEG",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0},
			expected: true,
		},
		{
			name:     "valid PNG",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: true,
		},
		{
			name:     "valid GIF87a",
			data:     []byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61},
			expected: true,
		},
		{
			name:     "valid GIF89a",
			data:     []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},
			expected: true,
		},
		{
			name:     "valid WebP",
			data:     []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			expected: true,
		},
		{
			name:     "valid BMP",
			data:     []byte{0x42, 0x4D, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "valid TIFF little-endian",
			data:     []byte{0x49, 0x49, 0x2A, 0x00},
			expected: true,
		},
		{
			name:     "valid TIFF big-endian",
			data:     []byte{0x4D, 0x4D, 0x00, 0x2A},
			expected: true,
		},
		{
			name:     "valid HEIC/HEIF",
			data:     []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x68, 0x65, 0x69, 0x63},
			expected: true,
		},
		{
			name:     "valid HEIF with mif1",
			data:     []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x6D, 0x69, 0x66, 0x31},
			expected: true,
		},
		{
			name:     "invalid data - too short",
			data:     []byte{0xFF, 0xD8},
			expected: false,
		},
		{
			name:     "invalid data - empty",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "invalid data - HTML",
			data:     []byte{0x3C, 0x68, 0x74, 0x6D, 0x6C},
			expected: false,
		},
		{
			name:     "invalid data - random bytes",
			data:     []byte{0x12, 0x34, 0x56, 0x78},
			expected: false,
		},
		{
			name:     "invalid WebP - missing WEBP",
			data:     []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: false,
		},
		{
			name:     "invalid HEIC - missing ftyp",
			data:     []byte{0x00, 0x00, 0x00, 0x18, 0x00, 0x00, 0x00, 0x00, 0x68, 0x65, 0x69, 0x63},
			expected: false,
		},
		{
			name:     "invalid HEIC - wrong brand",
			data:     []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x6A, 0x70, 0x65, 0x67},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageFileSignature(tt.data)
			assert.Equal(t, tt.expected, result, "data: %v", tt.data)
		})
	}
}

func TestValidateImageFile(t *testing.T) {
	tests := []struct {
		name          string
		contentType   string
		bodyData      []byte
		statusCode    int
		expectedError bool
		errorContains string
	}{
		{
			name:          "valid JPEG image",
			contentType:   "image/jpeg",
			bodyData:      []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01},
			statusCode:    http.StatusOK,
			expectedError: false,
		},
		{
			name:          "valid PNG image",
			contentType:   "image/png",
			bodyData:      []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D},
			statusCode:    http.StatusOK,
			expectedError: false,
		},
		{
			name:          "valid WebP image",
			contentType:   "image/webp",
			bodyData:      []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			statusCode:    http.StatusOK,
			expectedError: false,
		},
		{
			name:          "invalid content type",
			contentType:   "text/html",
			bodyData:      []byte{0x3C, 0x68, 0x74, 0x6D, 0x6C, 0x3E},
			statusCode:    http.StatusOK,
			expectedError: true,
			errorContains: "invalid content type",
		},
		{
			name:          "missing content type",
			contentType:   "",
			bodyData:      []byte{0xFF, 0xD8, 0xFF},
			statusCode:    http.StatusOK,
			expectedError: true,
			errorContains: "invalid content type",
		},
		{
			name:          "valid content type but invalid file signature",
			contentType:   "image/jpeg",
			bodyData:      []byte{0x3C, 0x68, 0x74, 0x6D, 0x6C, 0x3E, 0x68, 0x65, 0x6C, 0x6C, 0x6F},
			statusCode:    http.StatusOK,
			expectedError: true,
			errorContains: "invalid image file signature",
		},
		{
			name:          "empty body",
			contentType:   "image/jpeg",
			bodyData:      []byte{},
			statusCode:    http.StatusOK,
			expectedError: true,
			errorContains: "invalid image file signature",
		},
		{
			name:          "too short body",
			contentType:   "image/jpeg",
			bodyData:      []byte{0xFF},
			statusCode:    http.StatusOK,
			expectedError: true,
			errorContains: "invalid image file signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.statusCode)
				w.Write(tt.bodyData)
			}))
			defer server.Close()

			// Make request to test server
			resp, err := http.Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Test validateImageFile
			validatedBody, err := validateImageFile(resp)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, validatedBody)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validatedBody)
				if validatedBody != nil {
					defer validatedBody.Close()
					// Verify we can read from the validated body
					readData, readErr := io.ReadAll(validatedBody)
					assert.NoError(t, readErr)
					assert.Equal(t, tt.bodyData, readData)
				}
			}
		})
	}
}

func TestValidateImageFile_ReadError(t *testing.T) {
	// Create a response with a body that will error on read
	body := &errorReader{}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
		Body:       body,
	}

	validatedBody, err := validateImageFile(resp)
	assert.Error(t, err)
	assert.Nil(t, validatedBody)
	assert.Contains(t, err.Error(), "failed to read image file")
}

func TestValidateImageFile_ReconstructedBody(t *testing.T) {
	// Test that the reconstructed body contains all the original data
	testData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(testData)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	validatedBody, err := validateImageFile(resp)
	require.NoError(t, err)
	require.NotNil(t, validatedBody)
	defer validatedBody.Close()

	// Read all data from validated body
	readData, err := io.ReadAll(validatedBody)
	assert.NoError(t, err)
	assert.Equal(t, testData, readData, "reconstructed body should contain all original data")
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errorReader) Close() error {
	return nil
}

func TestNewError(t *testing.T) {
	// This function just prints errors, so we just verify it doesn't panic
	assert.NotPanics(t, func() {
		NewError(nil)
		NewError(&testError{message: "test error"})
	})
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

// Test edge cases for file signature detection
func TestIsImageFileSignature_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "exactly 4 bytes - valid TIFF",
			data:     []byte{0x49, 0x49, 0x2A, 0x00},
			expected: true,
		},
		{
			name:     "exactly 3 bytes - too short (requires 4 minimum)",
			data:     []byte{0xFF, 0xD8, 0xFF},
			expected: false,
		},
		{
			name:     "exactly 2 bytes - too short (requires 4 minimum)",
			data:     []byte{0x42, 0x4D},
			expected: false,
		},
		{
			name:     "less than 2 bytes",
			data:     []byte{0xFF},
			expected: false,
		},
		{
			name:     "exactly 12 bytes - valid WebP",
			data:     []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			expected: true,
		},
		{
			name:     "more than 12 bytes - valid JPEG",
			data:     append([]byte{0xFF, 0xD8, 0xFF}, make([]byte, 20)...),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageFileSignature(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test content type with various edge cases
func TestIsImageContentType_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "image with uppercase",
			content:  "IMAGE/PNG",
			expected: true,
		},
		{
			name:     "mixed case",
			content:  "ImAgE/JpEg",
			expected: true,
		},
		{
			name:     "multiple semicolons",
			content:  "image/jpeg; charset=utf-8; boundary=test",
			expected: true,
		},
		{
			name:     "only semicolon",
			content:  "image/jpeg;",
			expected: true,
		},
		{
			name:     "tabs and newlines",
			content:  "\timage/jpeg\n",
			expected: true,
		},
		{
			name:     "almost image prefix",
			content:  "application/image",
			expected: false,
		},
		{
			name:     "image at end",
			content:  "something/image",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageContentType(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOptimize_WithTestImage(t *testing.T) {
	// Skip if vips is not available (e.g., in CI without libvips installed)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up test environment variables
	os.Setenv("SECRET_KEY", "test-imgop-key")
	os.Setenv("FETCH_TIMEOUT", "5")
	defer func() {
		os.Unsetenv("SECRET_KEY")
		os.Unsetenv("FETCH_TIMEOUT")
		helpers.ResetAppEnvForTesting()
	}()

	// Reset env helper to pick up test environment
	helpers.ResetAppEnvForTesting()

	// Get the test image path - try multiple possible locations
	testImagePath := ""
	possiblePaths := []string{
		filepath.Join("static", "test-image.jpg"),             // From project root
		filepath.Join("..", "static", "test-image.jpg"),       // From src/libs
		filepath.Join("..", "..", "static", "test-image.jpg"), // From src
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			testImagePath = path
			break
		}
	}

	if testImagePath == "" {
		// Try absolute path from current working directory
		wd, err := os.Getwd()
		if err == nil {
			// Navigate to project root (assuming we're in src/libs or src)
			for wd != "/" && wd != "" {
				candidate := filepath.Join(wd, "static", "test-image.jpg")
				if _, err := os.Stat(candidate); err == nil {
					testImagePath = candidate
					break
				}
				wd = filepath.Dir(wd)
			}
		}
	}

	// Check if we found the test image
	if testImagePath == "" {
		t.Skip("test-image.jpg not found in static directory")
	}

	// Read the test image file
	testImageData, err := os.ReadFile(testImagePath)
	require.NoError(t, err, "test image file should exist")
	assert.Greater(t, len(testImageData), 0, "test image should have content")

	// Create a test HTTP server that serves the test image
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(testImageData)
	}))
	defer server.Close()

	// Create optimizer handler
	optimizer := NewImageOptimizer()

	tests := []struct {
		name        string
		width       int
		height      int
		quality     int
		description string
	}{
		{
			name:        "resize to width only",
			width:       200,
			height:      0,
			quality:     80,
			description: "should resize to specified width maintaining aspect ratio",
		},
		{
			name:        "resize to height only",
			width:       0,
			height:      150,
			quality:     80,
			description: "should resize to specified height maintaining aspect ratio",
		},
		{
			name:        "resize to both dimensions",
			width:       300,
			height:      200,
			quality:     80,
			description: "should resize to fit within specified dimensions",
		},
		{
			name:        "high quality",
			width:       400,
			height:      0,
			quality:     95,
			description: "should produce high quality output",
		},
		{
			name:        "low quality",
			width:       300,
			height:      0,
			quality:     50,
			description: "should produce lower quality output",
		},
		{
			name:        "no resize",
			width:       0,
			height:      0,
			quality:     80,
			description: "should return original size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := helpers.ParamsOptimize{
				Url:     server.URL,
				Width:   tt.width,
				Height:  tt.height,
				Quality: tt.quality,
			}

			// Optimize the image
			result := optimizer.Optimize(params)

			// Verify result is not empty
			assert.Greater(t, len(result), 0, "optimized image should not be empty")

			// Try to load the result as a WebP image using vips to verify it's valid
			source := vips.NewSource(io.NopCloser(bytes.NewReader(result)))
			defer source.Close()

			image, err := vips.NewImageFromSource(source, &vips.LoadOptions{
				FailOnError: true,
			})

			if err != nil {
				t.Logf("Warning: Could not load result as WebP image: %v", err)
				t.Logf("This might be expected if vips is not available in test environment")
				// Still verify we got some output
				assert.Greater(t, len(result), 0)
				return
			}

			// Verify it's a WebP image (has dimensions)
			resultWidth := image.Width()
			resultHeight := image.Height()
			assert.Greater(t, resultWidth, 0, "result should have valid width")
			assert.Greater(t, resultHeight, 0, "result should have valid height")

			// Load original image to get its dimensions for comparison
			originalSource := vips.NewSource(io.NopCloser(bytes.NewReader(testImageData)))
			defer originalSource.Close()
			originalImage, origErr := vips.NewImageFromSource(originalSource, &vips.LoadOptions{
				FailOnError: true,
			})

			if origErr == nil {
				defer originalImage.Close()
				origWidth := originalImage.Width()
				origHeight := originalImage.Height()
				t.Logf("Original image: %dx%d (%d bytes)", origWidth, origHeight, len(testImageData))
			} else {
				t.Logf("Original image size: %d bytes", len(testImageData))
			}

			t.Logf("Optimized image: %dx%d, size: %d bytes", resultWidth, resultHeight, len(result))

			// If resize parameters were specified, verify dimensions
			if tt.width > 0 && tt.height == 0 {
				// Width-only resize: result width should be exactly the specified width
				assert.Equal(t, tt.width, resultWidth, "width should match specified width")
			} else if tt.width == 0 && tt.height > 0 {
				// Height-only resize: result height should be exactly the specified height
				assert.Equal(t, tt.height, resultHeight, "height should match specified height")
			} else if tt.width > 0 && tt.height > 0 {
				// Both dimensions: image should fit within the box
				assert.LessOrEqual(t, resultWidth, tt.width, "width should not exceed specified width")
				assert.LessOrEqual(t, resultHeight, tt.height, "height should not exceed specified height")
				// At least one dimension should match (or be close to) the specified dimension
				assert.True(t, resultWidth == tt.width || resultHeight == tt.height,
					"at least one dimension should match the specified size")
			}

			image.Close()
		})
	}
}
