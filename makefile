deploy:
	@echo "🏗️  Building bootstrap binary..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-s -w" -o build/bootstrap src/main.go
	@echo "📦 Packaging for Lambda..."
	cd build && zip -j app.zip bootstrap
	@echo "✅ Created build/app.zip ($(shell du -h build/app.zip | cut -f1))"
	@echo ""
	@echo "⚠️  IMPORTANT: When deploying to Lambda:"
	@echo "   1. Upload build/libvips.zip as a Lambda Layer"
	@echo "   2. Attach the layer to your function"
	@echo "   3. Set environment variable: LD_LIBRARY_PATH=/opt/lib"
	@echo "   4. See DEPLOY_GUIDE.md for detailed instructions"

verify:
	./deployment-scripts/test/verify-build.sh

test-local:
	./deployment-scripts/test/test-local.sh

test-sam:
	./deployment-scripts/test/test-sam.sh

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
