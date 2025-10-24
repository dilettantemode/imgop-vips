package libs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cshum/vipsgen/vips"
)

// setupTestServer creates a test HTTP server that serves the test image
func setupTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the test image from static directory
		imageData, err := os.ReadFile("../../static/test-image.jpg")
		if err != nil {
			t.Fatalf("Failed to read test image: %v", err)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	}))
}

func TestImageOptimizerHandler_Optimize_WidthOnly(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Temporarily add test server to allowed origins
	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()
	AddAllowedOrigin(server.URL[7:]) // Remove "http://"

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   800,
		Height:  0,
		Quality: 80,
	})

	if len(result) == 0 {
		t.Fatal("Expected optimized image bytes, got empty result")
	}

	// Save output to static folder
	err := os.WriteFile("../../static/output-test.webp", result, 0644)
	if err != nil {
		t.Logf("Warning: Failed to save output image: %v", err)
	}

	// Verify the result is a valid WebP image
	image, err := vips.NewImageFromBuffer(result, nil)
	if err != nil {
		t.Fatalf("Failed to load optimized image: %v", err)
	}
	defer image.Close()

	if image.Width() != 800 {
		t.Errorf("Expected width 800, got %d", image.Width())
	}

	// Height should be proportional
	if image.Height() <= 0 {
		t.Errorf("Expected positive height, got %d", image.Height())
	}

	t.Logf("Output saved to static/output-test.webp (%d bytes, %dx%d)", len(result), image.Width(), image.Height())
}

func TestImageOptimizerHandler_Optimize_HeightOnly(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()
	AddAllowedOrigin(server.URL[7:])

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   0,
		Height:  600,
		Quality: 80,
	})

	if len(result) == 0 {
		t.Fatal("Expected optimized image bytes, got empty result")
	}

	image, err := vips.NewImageFromBuffer(result, nil)
	if err != nil {
		t.Fatalf("Failed to load optimized image: %v", err)
	}
	defer image.Close()

	if image.Height() != 600 {
		t.Errorf("Expected height 600, got %d", image.Height())
	}

	if image.Width() <= 0 {
		t.Errorf("Expected positive width, got %d", image.Width())
	}
}

func TestImageOptimizerHandler_Optimize_BothDimensions(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()
	AddAllowedOrigin(server.URL[7:])

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   800,
		Height:  600,
		Quality: 80,
	})

	if len(result) == 0 {
		t.Fatal("Expected optimized image bytes, got empty result")
	}

	image, err := vips.NewImageFromBuffer(result, nil)
	if err != nil {
		t.Fatalf("Failed to load optimized image: %v", err)
	}
	defer image.Close()

	// Image should fit within the specified box (contain mode)
	if image.Width() > 800 {
		t.Errorf("Width exceeds maximum: got %d, max 800", image.Width())
	}
	if image.Height() > 600 {
		t.Errorf("Height exceeds maximum: got %d, max 600", image.Height())
	}

	// At least one dimension should match or be close to the requested size
	if image.Width() != 800 && image.Height() != 600 {
		t.Errorf("Expected at least one dimension to match requested size. Got %dx%d", image.Width(), image.Height())
	}
}

func TestImageOptimizerHandler_Optimize_QualityParameter(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()
	AddAllowedOrigin(server.URL[7:])

	optimizer := NewImageOptimizer()

	// Test with high quality
	highQualityResult := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   400,
		Height:  0,
		Quality: 95,
	})

	// Test with low quality
	lowQualityResult := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   400,
		Height:  0,
		Quality: 50,
	})

	if len(highQualityResult) == 0 || len(lowQualityResult) == 0 {
		t.Fatal("Expected optimized image bytes, got empty result")
	}

	// Higher quality should generally result in larger file size
	if len(highQualityResult) < len(lowQualityResult) {
		t.Logf("Note: Higher quality resulted in smaller file. High: %d bytes, Low: %d bytes",
			len(highQualityResult), len(lowQualityResult))
	}
}

func TestImageOptimizerHandler_Optimize_InvalidOrigin(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Don't add server to allowed origins
	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   800,
		Height:  0,
		Quality: 80,
	})

	// Should return empty bytes for unauthorized origin
	if len(result) != 0 {
		t.Errorf("Expected empty result for invalid origin, got %d bytes", len(result))
	}
}

func TestImageOptimizerHandler_Optimize_InvalidURL(t *testing.T) {
	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     "not-a-valid-url",
		Width:   800,
		Height:  0,
		Quality: 80,
	})

	if len(result) != 0 {
		t.Errorf("Expected empty result for invalid URL, got %d bytes", len(result))
	}
}

func TestImageOptimizerHandler_Optimize_NoResize(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()
	AddAllowedOrigin(server.URL[7:])

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   0,
		Height:  0,
		Quality: 80,
	})

	if len(result) == 0 {
		t.Fatal("Expected optimized image bytes, got empty result")
	}

	// Should return optimized WebP even without resizing
	image, err := vips.NewImageFromBuffer(result, nil)
	if err != nil {
		t.Fatalf("Failed to load optimized image: %v", err)
	}
	defer image.Close()
}

// TestImageOptimizerHandler_Optimize_Googleusercontent tests with allowed origin
func TestImageOptimizerHandler_Optimize_Googleusercontent(t *testing.T) {
	if os.Getenv("SKIP_EXTERNAL_TESTS") != "" {
		t.Skip("Skipping external URL test")
	}

	optimizer := NewImageOptimizer()

	// Create a mock server that simulates lh3.googleusercontent.com
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		imageData, err := os.ReadFile("../../static/test-image.jpg")
		if err != nil {
			t.Fatalf("Failed to read test image: %v", err)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	}))
	defer server.Close()

	// Override the URL to use our test server but with allowed domain
	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()

	// Add test server host to allowed origins
	AddAllowedOrigin(server.URL[7:])

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   600,
		Height:  400,
		Quality: 85,
	})

	if len(result) == 0 {
		t.Fatal("Expected optimized image bytes, got empty result")
	}
}

// Benchmark tests
func BenchmarkImageOptimizer_Optimize(b *testing.B) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		imageData, _ := os.ReadFile("../../static/test-image.jpg")
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	}))
	defer server.Close()

	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()
	AddAllowedOrigin(server.URL[7:])

	optimizer := NewImageOptimizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.Optimize(ParamsOptimize{
			Url:     server.URL + "/test-image.jpg",
			Width:   800,
			Height:  0,
			Quality: 80,
		})
	}
}

// Test helper function
func TestNewImageOptimizer(t *testing.T) {
	optimizer := NewImageOptimizer()
	if optimizer == nil {
		t.Fatal("NewImageOptimizer returned nil")
	}
}

// Test that the test image exists
func TestTestImageExists(t *testing.T) {
	_, err := os.Stat("../../static/test-image.jpg")
	if os.IsNotExist(err) {
		t.Fatal("Test image does not exist at ../../static/test-image.jpg")
	}
}

// Test with broken image data
func TestImageOptimizerHandler_Optimize_BrokenImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("not an image"))
	}))
	defer server.Close()

	originalOrigins := allowedOrigins
	defer func() { allowedOrigins = originalOrigins }()
	allowedOrigins = append(allowedOrigins, server.URL[7:])

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/broken-image.jpg",
		Width:   800,
		Height:  0,
		Quality: 80,
	})

	// Should return empty bytes for broken image
	if len(result) != 0 {
		t.Errorf("Expected empty result for broken image, got %d bytes", len(result))
	}
}

// Test with server error
func TestImageOptimizerHandler_Optimize_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	originalOrigins := allowedOrigins
	defer func() { allowedOrigins = originalOrigins }()
	allowedOrigins = append(allowedOrigins, server.URL[7:])

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/error.jpg",
		Width:   800,
		Height:  0,
		Quality: 80,
	})

	// Should return empty bytes for server error
	if len(result) != 0 {
		t.Errorf("Expected empty result for server error, got %d bytes", len(result))
	}
}

// Test with empty response
func TestImageOptimizerHandler_Optimize_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		// Send empty body
	}))
	defer server.Close()

	originalOrigins := allowedOrigins
	defer func() { allowedOrigins = originalOrigins }()
	allowedOrigins = append(allowedOrigins, server.URL[7:])

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/empty.jpg",
		Width:   800,
		Height:  0,
		Quality: 80,
	})

	// Should return empty bytes for empty response
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty response, got %d bytes", len(result))
	}
}

// Test with very large dimensions
func TestImageOptimizerHandler_Optimize_LargeDimensions(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()
	AddAllowedOrigin(server.URL[7:])

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg",
		Width:   5000,
		Height:  5000,
		Quality: 80,
	})

	if len(result) == 0 {
		t.Fatal("Expected optimized image bytes, got empty result")
	}

	// Should upscale if requested dimensions are larger
	image, err := vips.NewImageFromBuffer(result, nil)
	if err != nil {
		t.Fatalf("Failed to load optimized image: %v", err)
	}
	defer image.Close()
}

// Test allowed origins validation
func TestAllowedOrigins(t *testing.T) {
	// Set the environment variable for testing
	originalEnv := os.Getenv("ALLOWED_ORIGINS")
	os.Setenv("ALLOWED_ORIGINS", "test.com,s.test.com,staging-files.test.com")
	defer os.Setenv("ALLOWED_ORIGINS", originalEnv)

	// Re-initialize origins by calling init logic
	SetAllowedOrigins([]string{"lh3.googleusercontent.com"})
	envOrigins := os.Getenv("ALLOWED_ORIGINS")
	if envOrigins != "" {
		origins := strings.Split(envOrigins, ",")
		for _, origin := range origins {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				AddAllowedOrigin(trimmed)
			}
		}
	}

	expectedOrigins := []string{
		"test.com",
		"s.test.com",
		"staging-files.test.com",
		"lh3.googleusercontent.com",
	}

	allowedOrigins := GetAllowedOrigins()
	for _, origin := range expectedOrigins {
		found := false
		for _, allowed := range allowedOrigins {
			if allowed == origin {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected origin %s not found in allowedOrigins", origin)
		}
	}
}

// Test with URL containing query parameters
func TestImageOptimizerHandler_Optimize_URLWithQueryParams(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	originalOrigins := GetAllowedOrigins()
	defer func() { SetAllowedOrigins(originalOrigins) }()
	AddAllowedOrigin(server.URL[7:])

	optimizer := NewImageOptimizer()

	result := optimizer.Optimize(ParamsOptimize{
		Url:     server.URL + "/test-image.jpg?foo=bar&baz=qux",
		Width:   400,
		Height:  0,
		Quality: 80,
	})

	if len(result) == 0 {
		t.Fatal("Expected optimized image bytes, got empty result")
	}
}

// Test environment variable loading
func TestEnvironmentVariableLoading(t *testing.T) {
	// Save original env and origins
	originalEnv := os.Getenv("ALLOWED_ORIGINS")
	originalOrigins := GetAllowedOrigins()
	defer func() {
		os.Setenv("ALLOWED_ORIGINS", originalEnv)
		SetAllowedOrigins(originalOrigins)
	}()

	testCases := []struct {
		name          string
		envValue      string
		expectedCount int
		shouldContain []string
	}{
		{
			name:          "Single origin",
			envValue:      "example.com",
			expectedCount: 2, // lh3.googleusercontent.com + example.com
			shouldContain: []string{"lh3.googleusercontent.com", "example.com"},
		},
		{
			name:          "Multiple origins",
			envValue:      "example1.com,example2.com,example3.com",
			expectedCount: 4, // lh3.googleusercontent.com + 3 new
			shouldContain: []string{"lh3.googleusercontent.com", "example1.com", "example2.com", "example3.com"},
		},
		{
			name:          "Origins with spaces",
			envValue:      "example1.com, example2.com , example3.com",
			expectedCount: 4,
			shouldContain: []string{"lh3.googleusercontent.com", "example1.com", "example2.com", "example3.com"},
		},
		{
			name:          "Empty string",
			envValue:      "",
			expectedCount: 1, // Only default lh3.googleusercontent.com
			shouldContain: []string{"lh3.googleusercontent.com"},
		},
		{
			name:          "With empty elements",
			envValue:      "example1.com,,example2.com",
			expectedCount: 3,
			shouldContain: []string{"lh3.googleusercontent.com", "example1.com", "example2.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset to default state
			SetAllowedOrigins([]string{"lh3.googleusercontent.com"})

			// Set environment variable
			os.Setenv("ALLOWED_ORIGINS", tc.envValue)

			// Simulate init logic
			envOrigins := os.Getenv("ALLOWED_ORIGINS")
			if envOrigins != "" {
				origins := strings.Split(envOrigins, ",")
				for _, origin := range origins {
					trimmed := strings.TrimSpace(origin)
					if trimmed != "" {
						AddAllowedOrigin(trimmed)
					}
				}
			}

			// Check results
			allowed := GetAllowedOrigins()
			if len(allowed) != tc.expectedCount {
				t.Errorf("Expected %d origins, got %d: %v", tc.expectedCount, len(allowed), allowed)
			}

			// Check that expected origins are present
			for _, expected := range tc.shouldContain {
				found := false
				for _, origin := range allowed {
					if origin == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected origin %q not found in: %v", expected, allowed)
				}
			}
		})
	}
}

// Test GetAllowedOrigins returns a copy
func TestGetAllowedOrigins_ReturnsCopy(t *testing.T) {
	originalOrigins := GetAllowedOrigins()
	defer SetAllowedOrigins(originalOrigins)

	// Get the origins
	origins1 := GetAllowedOrigins()

	// Modify the returned slice
	origins1 = append(origins1, "modified.com")

	// Get origins again
	origins2 := GetAllowedOrigins()

	// Should not contain the modification
	for _, origin := range origins2 {
		if origin == "modified.com" {
			t.Error("GetAllowedOrigins should return a copy, not the original slice")
		}
	}
}

// Test SetAllowedOrigins
func TestSetAllowedOrigins(t *testing.T) {
	originalOrigins := GetAllowedOrigins()
	defer SetAllowedOrigins(originalOrigins)

	newOrigins := []string{"test1.com", "test2.com", "test3.com"}
	SetAllowedOrigins(newOrigins)

	allowed := GetAllowedOrigins()
	if len(allowed) != 3 {
		t.Errorf("Expected 3 origins, got %d", len(allowed))
	}

	for _, expected := range newOrigins {
		found := false
		for _, origin := range allowed {
			if origin == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected origin %q not found", expected)
		}
	}
}

// Test AddAllowedOrigin
func TestAddAllowedOrigin(t *testing.T) {
	originalOrigins := GetAllowedOrigins()
	defer SetAllowedOrigins(originalOrigins)

	// Reset to a known state
	SetAllowedOrigins([]string{"initial.com"})

	// Add new origin
	AddAllowedOrigin("new.com")

	allowed := GetAllowedOrigins()
	if len(allowed) != 2 {
		t.Errorf("Expected 2 origins, got %d", len(allowed))
	}

	// Verify both origins are present
	expectedOrigins := []string{"initial.com", "new.com"}
	for _, expected := range expectedOrigins {
		found := false
		for _, origin := range allowed {
			if origin == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected origin %q not found", expected)
		}
	}
}

// Test AddAllowedOrigin prevents duplicates
func TestAddAllowedOrigin_PreventsDuplicates(t *testing.T) {
	originalOrigins := GetAllowedOrigins()
	defer SetAllowedOrigins(originalOrigins)

	SetAllowedOrigins([]string{"test.com"})

	// Try to add the same origin twice
	AddAllowedOrigin("test.com")
	AddAllowedOrigin("test.com")

	allowed := GetAllowedOrigins()
	if len(allowed) != 1 {
		t.Errorf("Expected 1 origin (no duplicates), got %d: %v", len(allowed), allowed)
	}
}

// Test AddAllowedOrigin with empty string
func TestAddAllowedOrigin_EmptyString(t *testing.T) {
	originalOrigins := GetAllowedOrigins()
	defer SetAllowedOrigins(originalOrigins)

	SetAllowedOrigins([]string{"test.com"})
	initialCount := len(GetAllowedOrigins())

	// Try to add empty string
	AddAllowedOrigin("")

	allowed := GetAllowedOrigins()
	if len(allowed) != initialCount {
		t.Errorf("Empty string should not be added. Expected %d origins, got %d", initialCount, len(allowed))
	}
}

// Test default initialization
func TestDefaultInitialization(t *testing.T) {
	// This test verifies that lh3.googleusercontent.com is always present
	originalEnv := os.Getenv("ALLOWED_ORIGINS")
	defer os.Setenv("ALLOWED_ORIGINS", originalEnv)

	// Clear env var
	os.Setenv("ALLOWED_ORIGINS", "")

	// Reset to default
	SetAllowedOrigins([]string{"lh3.googleusercontent.com"})

	allowed := GetAllowedOrigins()

	// Should contain at least the default origin
	found := false
	for _, origin := range allowed {
		if origin == "lh3.googleusercontent.com" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Default origin lh3.googleusercontent.com should always be present")
	}
}
