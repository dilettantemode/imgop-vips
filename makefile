deploy:
	go build -o build/bootstrap src/main.go
	./deployment-scripts/build.sh

dev:
	GOOS=linux GOARCH=amd64 go build -o build/bootstrap src/main.go
	@echo "Built for lambda-x86_64 (linux/amd64)"
	@echo "You can now test locally with AWS SAM CLI or run the bootstrap binary"

build-libvips:
	./deployment-scripts/build-libvips.sh

install:
	go mod tidy

# Upgrade mod
upgrade:
	go get -u
	go mod tidy
