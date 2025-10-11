build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap src/main.go

build-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap src/main.go

install:
	go mod tidy

# Upgrade mod
upgrade:
	go get -u
	go mod tidy

deploy:
	./deployment-scripts/build.sh