deploy:
	go build -o build/bootstrap src/main.go
	./deployment-scripts/build.sh

install:
	go mod tidy

# Upgrade mod
upgrade:
	go get -u
	go mod tidy
