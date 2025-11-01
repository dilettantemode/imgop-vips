package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type ParamsOptimize struct {
	Url     string
	Width   int
	Height  int
	Quality int
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func GetHeaders(headers map[string]string) map[string]string {
	headerData := make(map[string]string)
	for k, v := range headers {
		headerData[strings.ToLower(k)] = v
	}

	return headerData
}

func ParseParams[T int | string](reqParams map[string]string, key string) (T, error) {
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
	default:
		return any(value).(T), nil
	}
}

func ErrResponse(err error, statusCode int) (events.APIGatewayProxyResponse, error) {
	cacheControl := "public, max-age=259200, s-maxage=259200" // 3 days cache
	if statusCode == http.StatusForbidden {
		cacheControl = "public, max-age=60, s-maxage=60"
	}
	errorJSON, errJson := json.Marshal(ErrorResponse{
		Error: err.Error(),
	})

	if errJson != nil {
		errorJSON = []byte("{}")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       string(errorJSON),
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Cache-Control": cacheControl,
		},
	}, nil
}

func ValidateParams(params ParamsOptimize) (ParamsOptimize, error) {
	appEnv := GetAppEnv()
	imageParams := ParamsOptimize{
		Url:     params.Url,
		Width:   params.Width,
		Height:  params.Height,
		Quality: params.Quality,
	}

	if imageParams.Width < 0 || imageParams.Width > appEnv.MAX_WIDTH {
		return imageParams, fmt.Errorf("width must be between 0 and %d", appEnv.MAX_WIDTH)
	}
	if imageParams.Height < 0 || imageParams.Height > appEnv.MAX_HEIGHT {
		return imageParams, fmt.Errorf("height must be between 0 and %d", appEnv.MAX_HEIGHT)
	}
	if imageParams.Quality < 0 || imageParams.Quality > 100 {
		return imageParams, fmt.Errorf("quality must be between 0 and 100")
	}

	return imageParams, nil
}

func ValidateImage(params ParamsOptimize) (ParamsOptimize, error) {
	appEnv := GetAppEnv()
	imageParams := ParamsOptimize{
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

func IsAllowedOrigin(urlParam string) bool {
	appEnv := GetAppEnv()
	parsedUrl, err := url.Parse(urlParam)
	if err != nil {
		return false
	}

	origin := parsedUrl.Host
	if slices.Contains(appEnv.ALLOWED_ORIGINS, origin) {
		return true
	}
	return false
}
