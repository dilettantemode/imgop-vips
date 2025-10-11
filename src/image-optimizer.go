package imgop

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"

	"github.com/cshum/vipsgen/vips"
)

var allowedOrigins = []string{
	"tarkams.com",
	"s.tarkams.com",
	"staging-files.tarkams.com",
	"lh3.googleusercontent.com",
}

type ImageOptimizerHandler struct{}

func NewImageOptimizer() *ImageOptimizerHandler {
	return &ImageOptimizerHandler{}
}

type ParamsOptimize struct {
	Url     string
	Width   int
	Height  int
	Quality int
}

func (imgop *ImageOptimizerHandler) Optimize(params ParamsOptimize) []byte {
	// Validate if it is a proper url using simple reges
	imageUrl, err := url.Parse(params.Url)
	if err != nil {
		return []byte{}
	}

	domainName := imageUrl.Host
	if !slices.Contains(allowedOrigins, domainName) {
		return []byte{}
	}

	resp, err := http.Get(imageUrl.String())
	if err != nil {
		return []byte{}
	}
	defer resp.Body.Close()

	// Create source from io.ReadCloser
	source := vips.NewSource(resp.Body)
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

	var scale float64 = 1.0

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
