build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/bootstrap src/main.go
	./deployment-scripts/build.sh

build-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/bootstrap src/main.go
	./deployment-scripts/build.sh

install:
	go mod tidy

# Upgrade mod
upgrade:
	go get -u
	go mod tidy

deploy:
	./deployment-scripts/build.sh x86_64

deploy-arm:
	./deployment-scripts/build.sh arm64