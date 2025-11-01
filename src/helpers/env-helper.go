package helpers

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Singelton Env
type AppEnv struct {
	ALLOWED_ORIGINS []string
	SECRET_KEY      string
	MAX_WIDTH       int
	MAX_HEIGHT      int
	FETCH_TIMEOUT   int
}

var appEnv *AppEnv
var once sync.Once

// ResetAppEnvForTesting resets the singleton for testing purposes
// This should only be called in tests
func ResetAppEnvForTesting() {
	appEnv = nil
	once = sync.Once{}
}

func GetAppEnv() *AppEnv {
	once.Do(func() {
		allowedOrigins := []string{}
		if os.Getenv("ALLOWED_ORIGINS") != "" {
			allowedOrigins = strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
			for _, origin := range allowedOrigins {
				origin = strings.TrimSpace(origin)
				if origin != "" {
					allowedOrigins = append(allowedOrigins, origin)
				}
			}
		}

		secretKey := os.Getenv("SECRET_KEY")
		if secretKey == "" {
			log.Fatal("SECRET_KEY is not set")
		}

		maxWidth := 1800
		if maxWidthStr := os.Getenv("MAX_WIDTH"); maxWidthStr != "" {
			if mw, err := strconv.Atoi(maxWidthStr); err == nil && mw > 0 {
				maxWidth = mw
			}
		}
		maxHeight := 1800
		if maxHeightStr := os.Getenv("MAX_HEIGHT"); maxHeightStr != "" {
			if mh, err := strconv.Atoi(maxHeightStr); err == nil && mh > 0 {
				maxHeight = mh
			}
		}

		fetchTimeout := 5
		if fetchTimeoutStr := os.Getenv("FETCH_TIMEOUT"); fetchTimeoutStr != "" {
			if ft, err := strconv.Atoi(fetchTimeoutStr); err == nil && ft > 0 {
				fetchTimeout = ft
			}
		}

		appEnv = &AppEnv{
			ALLOWED_ORIGINS: allowedOrigins,
			SECRET_KEY:      os.Getenv("SECRET_KEY"),
			MAX_WIDTH:       maxWidth,
			MAX_HEIGHT:      maxHeight,
			FETCH_TIMEOUT:   fetchTimeout,
		}
	})
	return appEnv
}
