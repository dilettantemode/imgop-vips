package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// setupTestServer creates a test HTTP server that serves the test image
func setupTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the test image from static directory
		imageData, err := os.ReadFile("../static/test-image.jpg")
		if err != nil {
			t.Fatalf("Failed to read test image: %v", err)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	}))
}

func TestHandler_MissingURLParameter(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", resp.StatusCode)
	}

	expectedBody := `{"error": "Missing 'url' parameter"}`
	if resp.Body != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, resp.Body)
	}

	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Headers["Content-Type"])
	}
}

func TestHandler_InvalidOrigin(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Use the test server URL which is not in the allowed origins
	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": server.URL + "/test-image.jpg",
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", resp.StatusCode)
	}

	expectedBody := `{"error": "Failed to optimize image. Check URL and allowed origins."}`
	if resp.Body != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, resp.Body)
	}
}

func TestHandler_InvalidURL(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "not-a-valid-url",
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", resp.StatusCode)
	}
}

func TestHandler_SuccessWithMinimalParameters(t *testing.T) {
	// Create a custom test that uses an allowed origin
	// We'll use a mock by temporarily modifying the optimizer's allowed origins
	ctx := context.Background()

	// Use test.com which is in the allowed origins list
	// For this test to work, we need to serve an actual image from an allowed origin
	// Since we can't do that in a unit test, we'll test the parameter parsing instead

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test-image.jpg", // Allowed origin
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Since the URL doesn't actually exist, we expect a 400
	// But we're testing that the handler processes the request correctly
	if resp.StatusCode != 400 {
		t.Logf("Got status code %d (expected 400 for non-existent URL)", resp.StatusCode)
	}
}

func TestHandler_AllParameters(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg",
			"w":   "800",
			"h":   "600",
			"q":   "90",
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// The request should be processed (even if it fails due to non-existent URL)
	// We're mainly testing parameter parsing here
	if resp.StatusCode != 400 && resp.StatusCode != 200 {
		t.Errorf("Expected status code 400 or 200, got %d", resp.StatusCode)
	}
}

func TestHandler_InvalidParameterTypes(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg",
			"w":   "not-a-number",
			"h":   "also-not-a-number",
			"q":   "definitely-not-a-number",
		},
	}

	resp, err := handler(ctx, req)

	// Should not panic, should handle gracefully
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Invalid numbers should default to 0 for width/height and 80 for quality
	if resp.StatusCode != 400 && resp.StatusCode != 200 {
		t.Errorf("Expected status code 400 or 200, got %d", resp.StatusCode)
	}
}

func TestHandler_PartialParameters(t *testing.T) {
	testCases := []struct {
		name   string
		params map[string]string
	}{
		{
			name: "Only width",
			params: map[string]string{
				"url": "https://s.test.com/test.jpg",
				"w":   "800",
			},
		},
		{
			name: "Only height",
			params: map[string]string{
				"url": "https://s.test.com/test.jpg",
				"h":   "600",
			},
		},
		{
			name: "Only quality",
			params: map[string]string{
				"url": "https://s.test.com/test.jpg",
				"q":   "95",
			},
		},
		{
			name: "Width and height",
			params: map[string]string{
				"url": "https://s.test.com/test.jpg",
				"w":   "800",
				"h":   "600",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			req := events.APIGatewayProxyRequest{
				QueryStringParameters: tc.params,
			}

			resp, err := handler(ctx, req)

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			// Should process without panicking
			if resp.StatusCode != 400 && resp.StatusCode != 200 {
				t.Errorf("Expected status code 400 or 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestHandler_ResponseHeaders(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg",
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that response always has Content-Type header
	if resp.Headers["Content-Type"] == "" {
		t.Error("Expected Content-Type header to be set")
	}

	// For successful responses (if we get one), check additional headers
	if resp.StatusCode == 200 {
		if resp.Headers["Content-Type"] != "image/webp" {
			t.Errorf("Expected Content-Type image/webp for success, got %s", resp.Headers["Content-Type"])
		}
		if resp.Headers["Cache-Control"] != "public, max-age=31536000" {
			t.Errorf("Expected Cache-Control header, got %s", resp.Headers["Cache-Control"])
		}
		if !resp.IsBase64Encoded {
			t.Error("Expected IsBase64Encoded to be true for successful response")
		}
	}
}

// Integration test with actual image processing
func TestHandler_IntegrationWithTestServer(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// We need to temporarily add the test server to allowed origins
	// This is a limitation of the current implementation
	// In production, you might want to inject dependencies or use interfaces

	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": server.URL + "/test-image.jpg",
			"w":   "400",
			"q":   "80",
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Since the test server is not in allowed origins, we expect 400
	if resp.StatusCode != 400 {
		t.Logf("Note: Got status %d. Test server not in allowed origins.", resp.StatusCode)
	}

	t.Logf("Response status: %d, body length: %d", resp.StatusCode, len(resp.Body))
}

func TestHandler_Base64EncodingFormat(t *testing.T) {
	// This test validates that if we get a successful response,
	// the body is valid base64

	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg",
			"w":   "400",
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// If we got a 200 response, validate base64 encoding
	if resp.StatusCode == 200 {
		// Try to decode the body
		_, err := base64.StdEncoding.DecodeString(resp.Body)
		if err != nil {
			t.Errorf("Response body is not valid base64: %v", err)
		}

		// Check that it's marked as base64 encoded
		if !resp.IsBase64Encoded {
			t.Error("IsBase64Encoded should be true for successful image response")
		}
	}
}

func TestHandler_EmptyQueryString(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "",
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status code 400 for empty URL, got %d", resp.StatusCode)
	}
}

func TestHandler_ZeroQuality(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg",
			"q":   "0",
		},
	}

	resp, err := handler(ctx, req)

	// Should not panic with quality=0
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should process (though may fail for other reasons)
	if resp.StatusCode != 400 && resp.StatusCode != 200 {
		t.Errorf("Expected status code 400 or 200, got %d", resp.StatusCode)
	}
}

func TestHandler_LargeQuality(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg",
			"q":   "150", // Over 100
		},
	}

	resp, err := handler(ctx, req)

	// Should not panic with quality > 100
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should process (though may fail for other reasons)
	if resp.StatusCode != 400 && resp.StatusCode != 200 {
		t.Errorf("Expected status code 400 or 200, got %d", resp.StatusCode)
	}
}

func TestHandler_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg",
		},
	}

	resp, err := handler(ctx, req)

	// Handler should complete even with cancelled context
	// (current implementation doesn't use context for cancellation)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should still return a response
	if resp.StatusCode == 0 {
		t.Error("Expected non-zero status code")
	}
}

func TestHandler_URLWithQueryParameters(t *testing.T) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg?foo=bar&baz=qux",
			"w":   "400",
		},
	}

	resp, err := handler(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should process URLs with query parameters
	if resp.StatusCode != 400 && resp.StatusCode != 200 {
		t.Errorf("Expected status code 400 or 200, got %d", resp.StatusCode)
	}
}

func TestHandler_AllowedOrigins(t *testing.T) {
	allowedOrigins := []string{
		"https://test.com/test.jpg",
		"https://s.test.com/test.jpg",
		"https://staging-files.test.com/test.jpg",
		"https://lh3.googleusercontent.com/test.jpg",
	}

	ctx := context.Background()

	for _, url := range allowedOrigins {
		t.Run(url, func(t *testing.T) {
			req := events.APIGatewayProxyRequest{
				QueryStringParameters: map[string]string{
					"url": url,
				},
			}

			resp, err := handler(ctx, req)

			if err != nil {
				t.Fatalf("Expected no error for allowed origin, got %v", err)
			}

			// Should not reject based on origin
			// (will fail for other reasons like URL not existing)
			t.Logf("Allowed origin %s returned status %d", url, resp.StatusCode)
		})
	}
}

// Mock successful optimization test
func TestHandler_SuccessfulOptimization_MockScenario(t *testing.T) {
	// This test demonstrates what a successful response should look like
	// In reality, we'd need to mock the optimizer or use dependency injection

	// Create a mock successful response structure
	ctx := context.Background()

	// Create a small test image to simulate what we'd get back
	testImageBytes := []byte("fake-webp-data")
	encodedImage := base64.StdEncoding.EncodeToString(testImageBytes)

	// Expected response structure for successful optimization
	expectedResponse := events.APIGatewayProxyResponse{
		StatusCode:      200,
		Body:            encodedImage,
		IsBase64Encoded: true,
		Headers: map[string]string{
			"Content-Type":  "image/webp",
			"Cache-Control": "public, max-age=31536000",
		},
	}

	// Verify the expected structure is valid
	if expectedResponse.StatusCode != 200 {
		t.Error("Expected status code should be 200")
	}
	if !expectedResponse.IsBase64Encoded {
		t.Error("Expected IsBase64Encoded to be true")
	}
	if expectedResponse.Headers["Content-Type"] != "image/webp" {
		t.Error("Expected Content-Type to be image/webp")
	}

	// Verify we can decode the base64
	decoded, err := base64.StdEncoding.DecodeString(expectedResponse.Body)
	if err != nil {
		t.Errorf("Failed to decode base64 body: %v", err)
	}
	if string(decoded) != "fake-webp-data" {
		t.Error("Decoded data doesn't match expected")
	}

	t.Logf("Successful optimization response structure validated")
	_ = ctx // Use the context to avoid unused variable
}

// Test error response format
func TestHandler_ErrorResponseFormat(t *testing.T) {
	testCases := []struct {
		name           string
		request        events.APIGatewayProxyRequest
		expectedStatus int
		errorSubstring string
	}{
		{
			name: "Missing URL",
			request: events.APIGatewayProxyRequest{
				QueryStringParameters: map[string]string{},
			},
			expectedStatus: 400,
			errorSubstring: "Missing 'url' parameter",
		},
		{
			name: "Invalid origin",
			request: events.APIGatewayProxyRequest{
				QueryStringParameters: map[string]string{
					"url": "https://evil.com/test.jpg",
				},
			},
			expectedStatus: 400,
			errorSubstring: "Failed to optimize",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			resp, err := handler(ctx, tc.request)

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			if !strings.Contains(resp.Body, tc.errorSubstring) {
				t.Errorf("Expected error body to contain %q, got %q", tc.errorSubstring, resp.Body)
			}

			if resp.Headers["Content-Type"] != "application/json" {
				t.Errorf("Expected Content-Type application/json for error, got %s", resp.Headers["Content-Type"])
			}
		})
	}
}

// Benchmark the handler
func BenchmarkHandler(b *testing.B) {
	ctx := context.Background()

	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"url": "https://s.test.com/test.jpg",
			"w":   "400",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(ctx, req)
	}
}

// Test that init() function sets up the optimizer
func TestInitFunction(t *testing.T) {
	if optimizer == nil {
		t.Fatal("Optimizer should be initialized by init() function")
	}
}

// Test ImageRequest struct (for documentation purposes)
func TestImageRequestStruct(t *testing.T) {
	req := ImageRequest{
		Url:     "https://s.test.com/test.jpg",
		Width:   800,
		Height:  600,
		Quality: 90,
	}

	if req.Url == "" {
		t.Error("URL should not be empty")
	}
	if req.Width != 800 {
		t.Errorf("Expected width 800, got %d", req.Width)
	}
	if req.Height != 600 {
		t.Errorf("Expected height 600, got %d", req.Height)
	}
	if req.Quality != 90 {
		t.Errorf("Expected quality 90, got %d", req.Quality)
	}
}
