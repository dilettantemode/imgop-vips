package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

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
	reqHeaders := helpers.GetHeaders(req.Headers)
	authHeader, ok := reqHeaders["imgop-key"]
	if !ok || authHeader != appEnv.SECRET_KEY {
		return helpers.ErrResponse(fmt.Errorf("Forbidden, secret key is incorrect"), http.StatusForbidden)
	}

	qParams := req.QueryStringParameters
	width, err1 := helpers.ParseParams[int](qParams, "w")
	height, err2 := helpers.ParseParams[int](qParams, "h")
	quality, _ := helpers.ParseParams[int](qParams, "q")

	if width+height == 0 {
		if err1 != nil {
			return helpers.ErrResponse(err1, http.StatusUnprocessableEntity)
		}
		if err2 != nil {
			return helpers.ErrResponse(err2, http.StatusUnprocessableEntity)
		}
	}

	urlParams, err4 := helpers.ParseParams[string](qParams, "url")
	if err4 != nil {
		return helpers.ErrResponse(err4, http.StatusUnprocessableEntity)
	}
	urlParams, err5 := url.QueryUnescape(urlParams)
	if err5 != nil {
		return helpers.ErrResponse(err5, http.StatusUnprocessableEntity)
	}
	isValidUrl := helpers.IsAllowedOrigin(urlParams)
	if !isValidUrl {
		return helpers.ErrResponse(fmt.Errorf("invalid url allowed origin"), http.StatusUnprocessableEntity)
	}

	imageParams := helpers.ParamsOptimize{
		Url:     urlParams,
		Width:   width,
		Height:  height,
		Quality: quality,
	}

	imageParams, errImg := helpers.ValidateParams(imageParams)
	if errImg != nil {
		return helpers.ErrResponse(errImg, http.StatusUnprocessableEntity)
	}

	imageBytes := optimizer.Optimize(imageParams)
	cacheTime := "31536000" // 1 year cache
	return events.APIGatewayProxyResponse{
		StatusCode:      http.StatusOK,
		Body:            base64.StdEncoding.EncodeToString(imageBytes),
		IsBase64Encoded: true,
		Headers: map[string]string{
			"Content-Type":  "image/webp",
			"Cache-Control": "public, max-age=" + cacheTime + ", s-maxage=" + cacheTime, // 1 year cache
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
