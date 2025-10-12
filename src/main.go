package imgop

import (
	"context"
	"encoding/base64"
	"fmt"

	libs "imgop/src/libs"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type ImageRequest struct {
	Url     string `json:"url"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Quality int    `json:"quality,omitempty"`
}

var optimizer *libs.ImageOptimizerHandler

func init() {
	optimizer = libs.NewImageOptimizer()
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse query parameters
	url := req.QueryStringParameters["url"]
	if url == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "Missing 'url' parameter"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	// Parse optional parameters with defaults
	width := 0
	height := 0
	quality := 80

	if w := req.QueryStringParameters["w"]; w != "" {
		fmt.Sscanf(w, "%d", &width)
	}
	if h := req.QueryStringParameters["h"]; h != "" {
		fmt.Sscanf(h, "%d", &height)
	}
	if q := req.QueryStringParameters["q"]; q != "" {
		fmt.Sscanf(q, "%d", &quality)
	}

	// Call the optimizer
	imageBytes := optimizer.Optimize(libs.ParamsOptimize{
		Url:     url,
		Width:   width,
		Height:  height,
		Quality: quality,
	})

	if len(imageBytes) == 0 {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "Failed to optimize image. Check URL and allowed origins."}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}

	// Return base64-encoded image for API Gateway
	encodedImage := base64.StdEncoding.EncodeToString(imageBytes)

	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Body:            encodedImage,
		IsBase64Encoded: true,
		Headers: map[string]string{
			"Content-Type":  "image/webp",
			"Cache-Control": "public, max-age=31536000",
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
