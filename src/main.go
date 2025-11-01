package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"

	"imgop/src/helpers"
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
	// Check authentication
	appEnv := helpers.GetAppEnv()
	authHeader := req.Headers["secret-auth-key"]
	if authHeader != appEnv.SECRET_KEY {
		return errResponse(fmt.Errorf("Forbidden, secret key is incorrect"), http.StatusForbidden)
	}

	width, err1 := parseParams[int](req.QueryStringParameters, "w")
	if err1 != nil {
		return errResponse(err1, http.StatusUnprocessableEntity)
	}
	height, err2 := parseParams[int](req.QueryStringParameters, "h")
	if err2 != nil {
		return errResponse(err2, http.StatusUnprocessableEntity)
	}
	quality, err3 := parseParams[int](req.QueryStringParameters, "q")
	if err3 != nil {
		return errResponse(err3, http.StatusUnprocessableEntity)
	}
	urlParams, err4 := parseParams[string](req.QueryStringParameters, "url")
	if err4 != nil {
		return errResponse(err4, http.StatusUnprocessableEntity)
	}
	// Get and validate query parameters
	imageParams := libs.ParamsOptimize{
		Url:     urlParams,
		Width:   width,
		Height:  height,
		Quality: quality,
	}

	imageParams, err := validateParams(imageParams)
	if err != nil {
		return errResponse(err, http.StatusUnprocessableEntity)
	}

	imageBytes := optimizer.Optimize(imageParams)
	cacheTime := "31536000" // 1 year cache
	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Body:            base64.StdEncoding.EncodeToString(imageBytes),
		IsBase64Encoded: true,
		Headers: map[string]string{
			"Content-Type":  "image/webp",
			"Cache-Control": "public, max-age=" + cacheTime + ", s-maxage=" + cacheTime, // 1 year cache
		},
	}, nil
}

func parseParams[T int | string](reqParams map[string]string, key string) (T, error) {
	var zero T
	value, ok := reqParams[key]
	if !ok {
		return zero, fmt.Errorf("missing %s parameter", key)
	}

	switch any(zero).(type) {
	case int:
		if val, err := strconv.Atoi(value); err == nil {
			return any(val).(T), nil
		}
		return zero, fmt.Errorf("invalid integer value for %s parameter", key)
	case string:
		return any(value).(T), nil
	}

	return zero, nil
}

func errResponse(err error, statusCode int) (events.APIGatewayProxyResponse, error) {
	cacheControl := "public, max-age=259200, s-maxage=259200" // 3 days cache
	if statusCode == http.StatusForbidden {
		cacheControl = "public, max-age=60, s-maxage=60"
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       fmt.Sprintf(`{"error": "%s"}`, err.Error()),
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Cache-Control": cacheControl,
		},
	}, nil
}

func validateParams(params libs.ParamsOptimize) (libs.ParamsOptimize, error) {
	appEnv := helpers.GetAppEnv()
	imageParams := libs.ParamsOptimize{
		Url:     params.Url,
		Width:   params.Width,
		Height:  params.Height,
		Quality: params.Quality,
	}

	if imageParams.Width < 1 || imageParams.Width > appEnv.MAX_WIDTH {
		return imageParams, fmt.Errorf("width must be between 1 and %d", appEnv.MAX_WIDTH)
	}
	if imageParams.Height < 1 || imageParams.Height > appEnv.MAX_HEIGHT {
		return imageParams, fmt.Errorf("height must be between 1 and %d", appEnv.MAX_HEIGHT)
	}
	if imageParams.Quality < 1 || imageParams.Quality > 100 {
		return imageParams, fmt.Errorf("quality must be between 1 and 100")
	}

	return imageParams, nil
}

func main() {
	lambda.Start(handler)
}
