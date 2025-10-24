# Image Operations Lambda (imgop)

A serverless image optimization service running on AWS Lambda that converts and optimizes images to WebP format with customizable dimensions and quality settings.

## Table of Contents

- [Quick Start](#quick-start)
- [Requirements](#requirements)
- [Project Structure](#project-structure)
- [Local Development](#local-development)
- [Building for Lambda](#building-for-lambda)
- [AWS Lambda Deployment](#aws-lambda-deployment)
- [Testing & Verification](#testing--verification)
- [Troubleshooting](#troubleshooting)
- [Makefile Commands](#makefile-commands)
- [Architecture](#architecture)
- [Security & Performance](#security--performance)
- [Important Notes](#important-notes)

## Quick Start

```bash
# 1. Build the Lambda function and libvips layer
make deploy                  # Creates build/app.zip
make build-libvips          # Creates build/libvips.zip

# 2. Verify build artifacts
make verify

# 3. Test locally (optional)
make test-sam               # Using AWS SAM CLI
# OR
make test-local             # Using Docker

# 4. Deploy to AWS (see deployment section below)
# - Upload build/libvips.zip as Lambda Layer
# - Deploy build/app.zip as Lambda function
# - Attach layer and set LD_LIBRARY_PATH=/opt/lib
```

**⚠️ Critical:** The bootstrap binary requires libvips libraries at runtime. You **must**:
1. Deploy `build/libvips.zip` as a Lambda Layer
2. Attach the layer to your Lambda function
3. Set environment variable: `LD_LIBRARY_PATH=/opt/lib`

## Requirements

- Go 1.21+ installed
- Docker (for building libvips layer)
- AWS CLI configured with appropriate credentials
- AWS Lambda runtime environment
- libvips 8.17.2 for Lambda

## Project Structure

```
imgop/
├── src/                         # Source code
│   ├── main.go                 # Lambda handler (package main)
│   ├── main_test.go            # Tests
│   └── libs/                   # Image optimization libraries
│       ├── image-optimizer.go
│       └── image-optimizer_test.go
│
├── deployment-scripts/          # All deployment and testing infrastructure
│   ├── docker/                 # Docker files
│   │   ├── Dockerfile.build-vips
│   │   ├── Dockerfile.lambda-build
│   │   └── Dockerfile.test
│   ├── test/                   # Testing and verification
│   │   ├── template.yaml
│   │   ├── test-event.json
│   │   ├── test-sam.sh
│   │   ├── test-local.sh
│   │   └── verify-build.sh
│   ├── ec2/                    # EC2 deployment (if needed)
│   ├── build-libvips.sh
│   └── build.sh
│
├── build/                       # Build artifacts (gitignored)
│   ├── bootstrap               # Lambda function binary
│   ├── app.zip                 # Deployment package (~3.4 MB)
│   └── libvips.zip             # Lambda layer package (~5.0 MB)
│
├── docs/                        # API documentation
├── static/                      # Static assets for testing
├── go.mod                       # Go module definition
├── go.sum                       # Go dependencies
├── makefile                     # Build automation
└── README.md                    # This file
```

### Key Files

**Source Code:**
- `src/main.go` - Lambda handler function (must use `package main`)
- `src/libs/image-optimizer.go` - Core image optimization logic using libvips

**Build Configuration:**
- `makefile` - Automation for build, test, and deployment tasks
- `go.mod` - Go module dependencies

**Deployment:**
- `deployment-scripts/docker/Dockerfile.build-vips` - Builds libvips Lambda layer
- `deployment-scripts/build-libvips.sh` - Executes libvips layer build

**Testing:**
- `deployment-scripts/test/verify-build.sh` - Verifies build artifacts
- `deployment-scripts/test/test-sam.sh` - Tests with AWS SAM CLI
- `deployment-scripts/test/test-local.sh` - Tests with Docker

## Local Development

### 1. Install Dependencies

```bash
make install
```

### 2. Build for Development

```bash
make dev
```

This builds the Lambda function for x86_64 architecture (linux/amd64) and outputs to `build/bootstrap`.

### 3. Local Testing

#### Option A: Using AWS SAM CLI

```bash
# Install SAM CLI if you haven't already
# https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html

# Run test
make test-sam
```

#### Option B: Using Docker

```bash
# Run automated Docker test
make test-local
```

#### Option C: Manual Testing with RIE

```bash
# Run the built binary with AWS Lambda Runtime Interface Emulator
docker run -p 9000:8080 -v $(pwd)/build:/var/task \
  amazon/aws-lambda-runtime-interface-emulator ./bootstrap
```

## Building for Lambda

### Build the Application

```bash
make deploy
```

This command:
1. Compiles the Go application for Linux/AMD64 with CGO enabled
2. Creates `build/bootstrap` executable (ELF 64-bit)
3. Packages it into `build/app.zip` (~3.4 MB)

### Build libvips Layer

The application requires libvips 8.17.2 at runtime. Build the Lambda layer:

```bash
make build-libvips
```

This command:
1. Uses Docker to build libvips 8.17.2 in an Alpine Linux environment
2. Compiles libvips with required dependencies (libjpeg-turbo, libpng, libwebp, libexif, etc.)
3. Packages binaries and libraries into `build/libvips.zip` (~5.0 MB)

**Note:** This process takes several minutes. Run it once unless you need to update libvips.

### Verify Build Artifacts

```bash
make verify
```

This checks:
- ✅ bootstrap is ELF 64-bit executable
- ✅ app.zip exists and contains bootstrap
- ✅ libvips.zip exists and contains shared libraries (.so files)
- ✅ bootstrap binary links to libvips

## AWS Lambda Deployment

### Step 1: Create Lambda Layer

Upload the libvips layer to AWS Lambda:

**Via AWS Console:**
1. Go to AWS Lambda → Layers
2. Click "Create layer"
3. Name: `libvips-runtime`
4. Upload: `build/libvips.zip`
5. Compatible runtimes: `Custom runtime on Amazon Linux 2` (provided.al2)
6. Compatible architectures: `x86_64`
7. Click "Create"

**Via AWS CLI:**
```bash
aws lambda publish-layer-version \
    --layer-name libvips-runtime \
    --description "libvips 8.17.2 runtime libraries" \
    --zip-file fileb://build/libvips.zip \
    --compatible-runtimes provided.al2 \
    --compatible-architectures x86_64
```

**Save the Layer ARN** (e.g., `arn:aws:lambda:us-east-1:123456789012:layer:libvips-runtime:1`)

### Step 2: Create Lambda Function

**Via AWS Console:**

1. Navigate to AWS Lambda → Functions → Create function
2. Choose "Author from scratch"
3. Configuration:
   - **Function name:** `imgop` (or your preferred name)
   - **Runtime:** `Custom runtime on Amazon Linux 2` (provided.al2)
   - **Architecture:** `x86_64`
   - **Execution role:** Create new role or use existing with basic Lambda permissions
4. Click "Create function"

5. Upload function code:
   - In the "Code" tab, click "Upload from" → ".zip file"
   - Upload `build/app.zip`
   - Click "Save"

6. Configure function settings:
   - **Handler:** `bootstrap` (default for custom runtimes)
   - **Memory:** 512 MB (minimum recommended, adjust based on image sizes)
   - **Timeout:** 30 seconds (adjust based on your needs)
   - **Ephemeral storage:** 512 MB (default is fine)

**Via AWS CLI:**

```bash
# Create the function
aws lambda create-function \
    --function-name imgop \
    --runtime provided.al2 \
    --role arn:aws:iam::YOUR_ACCOUNT_ID:role/YOUR_LAMBDA_ROLE \
    --handler bootstrap \
    --zip-file fileb://build/app.zip \
    --timeout 30 \
    --memory-size 512 \
    --architectures x86_64
```

### Step 3: Attach Layer and Configure Environment

**Add the Layer:**

Via Console:
1. Open your Lambda function
2. Scroll to "Layers" → Click "Add a layer"
3. Choose "Custom layers"
4. Select `libvips-runtime` layer
5. Click "Add"

Via CLI:
```bash
aws lambda update-function-configuration \
    --function-name imgop \
    --layers arn:aws:lambda:REGION:ACCOUNT:layer:libvips-runtime:1
```

**⚠️ Critical: Configure Environment Variables**

Lambda needs to know where to find the libraries:

Via Console:
1. Configuration tab → Environment variables → Edit
2. Add:
   - `LD_LIBRARY_PATH` = `/opt/lib`
   - `PATH` = `/opt/bin:$PATH`

Via CLI:
```bash
aws lambda update-function-configuration \
    --function-name imgop \
    --environment "Variables={LD_LIBRARY_PATH=/opt/lib,PATH=/opt/bin:/usr/local/bin:/usr/bin:/bin}"
```

**Why this is critical:** Without `LD_LIBRARY_PATH=/opt/lib`, Lambda will exit with error 127 (library not found).

### Step 4: Set Up API Gateway

To make your Lambda function accessible via HTTP/HTTPS:

**Via AWS Console:**

1. Navigate to API Gateway → Create API
2. Choose "HTTP API" (simpler and cheaper) or "REST API" (more features)
3. Click "Build"
4. Configuration:
   - **API name:** `imgop-api`
   - **Integration:** AWS Lambda
   - **Lambda function:** Select your `imgop` function
5. Configure routes:
   - **Method:** `GET`
   - **Resource path:** `/` or `/optimize`
6. Configure stages:
   - **Stage name:** `prod` or `$default`
7. Create and deploy
8. Note the Invoke URL

**Via AWS CLI:**

```bash
# Create HTTP API
aws apigatewayv2 create-api \
    --name imgop-api \
    --protocol-type HTTP \
    --target arn:aws:lambda:REGION:ACCOUNT:function:imgop

# Grant API Gateway permission to invoke Lambda
aws lambda add-permission \
    --function-name imgop \
    --statement-id apigateway-invoke \
    --action lambda:InvokeFunction \
    --principal apigatewayv2.amazonaws.com
```

### Step 5: Test Your Deployment

```bash
# Test with curl
curl "https://YOUR_API_GATEWAY_URL/?url=https://via.placeholder.com/1200x800.jpg&w=800&q=85" \
    --output optimized.webp
```

**API Parameters:**
- `url` (required): URL of the image to optimize
- `w` (optional): Target width in pixels
- `h` (optional): Target height in pixels
- `q` (optional): Quality (1-100, default: 80)

**Example:**
```bash
curl "https://abc123.execute-api.us-east-1.amazonaws.com/?url=https://example.com/sample.jpg&w=1200&h=800&q=90" \
    --output optimized.webp
```

### Step 6: Update Existing Deployment

When you make code changes:

```bash
# Build new version
make deploy

# Update Lambda function code
aws lambda update-function-code \
    --function-name imgop \
    --zip-file fileb://build/app.zip

# Or via AWS Console: Upload new app.zip in the Code tab
```

## Testing & Verification

### Verify Build Artifacts

```bash
make verify
```

Checks:
- Bootstrap is ELF 64-bit executable
- app.zip contains bootstrap
- libvips.zip contains shared libraries
- Bootstrap links to libvips

### Local Testing with AWS SAM CLI

```bash
make test-sam
```

Requirements: AWS SAM CLI installed

### Local Testing with Docker

```bash
make test-local
```

Requirements: Docker running

### Manual Testing

```bash
# Navigate to test directory
cd deployment-scripts/test

# Run individual test scripts
./verify-build.sh
./test-sam.sh
./test-local.sh
```

## Troubleshooting

### Exit Status 127 Error

**Problem:** Lambda returns "Runtime.ExitError: exit status 127"

**Cause:** The dynamic linker can't find required shared libraries.

**Solution:**
1. Verify the libvips layer is attached to your function
2. **Critical:** Check environment variable `LD_LIBRARY_PATH=/opt/lib` is set
3. Verify libvips.zip contains `.so` files:
   ```bash
   unzip -l build/libvips.zip | grep ".so"
   ```

### Binary Format Error (exec format error)

**Problem:** Lambda returns "Runtime.InvalidEntrypoint: exec format error"

**Cause:** The bootstrap binary was compiled for the wrong architecture or isn't a proper executable.

**Solution:**
1. Ensure you're building with correct flags:
   ```bash
   GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o build/bootstrap src/main.go
   ```
2. Verify the binary is ELF 64-bit:
   ```bash
   file build/bootstrap
   # Should output: ELF 64-bit LSB executable, x86-64
   ```
3. Check that `src/main.go` uses `package main` (not `package imgop`)

### libvips Not Found

**Problem:** Lambda function fails with "libvips not found" or similar errors

**Solution:**
- Verify the libvips layer is attached to your function
- Check that the layer was built for the correct architecture (x86_64)
- Ensure environment variable `LD_LIBRARY_PATH=/opt/lib` is set
- Verify layer structure:
  ```bash
  unzip -l build/libvips.zip
  # Should show lib/ directory with .so files
  ```

### Image Optimization Fails

**Problem:** Function returns 400 error

**Solution:**
- Verify the source URL is accessible from Lambda
- Check image format is supported (JPEG, PNG, WebP, GIF)
- Ensure sufficient memory allocation (min 512 MB)
- Check CloudWatch Logs for detailed error messages

### Timeout Errors

**Problem:** Function times out

**Solution:**
- Increase timeout setting in Lambda configuration
- Increase memory allocation (more memory = faster CPU)
- Check if source image is too large
- Monitor CloudWatch Logs for slow operations

### Library Compatibility Issues

**Problem:** Binary links against system libraries not available in Lambda

**Solution:**
- Ensure you're building the libvips layer with the provided Dockerfile
- The layer should include all required `.so` files
- Consider building in a Lambda-compatible environment (Amazon Linux 2)

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make install` | Install/update Go dependencies |
| `make dev` | Build for local development (lambda-x86_64) |
| `make deploy` | Build application and package for Lambda deployment |
| `make build-libvips` | Build libvips layer for Lambda (requires Docker) |
| `make verify` | Verify build artifacts are correct |
| `make test-sam` | Test locally using AWS SAM CLI |
| `make test-local` | Test locally using Docker |
| `make upgrade` | Upgrade Go module dependencies |

## Architecture

### How It Works

This Lambda function:
1. Receives image URL and optimization parameters via API Gateway
2. Downloads the image from the provided URL
3. Uses libvips to resize/optimize the image
4. Converts to WebP format
5. Returns the optimized image as base64-encoded response
6. API Gateway decodes and serves the binary image

### Lambda Function Configuration

**Recommended Settings:**

| Setting | Value | Notes |
|---------|-------|-------|
| Memory | 512 MB - 1024 MB | Adjust based on image sizes |
| Timeout | 30 seconds | Adjust for larger images |
| Ephemeral storage | 512 MB | Default is sufficient |
| Architecture | x86_64 | Must match libvips build |
| Runtime | provided.al2 | Custom Go runtime |

**Required IAM Permissions:**

Your Lambda execution role needs:
- `logs:CreateLogGroup`
- `logs:CreateLogStream`
- `logs:PutLogEvents`
- (Optional) S3 permissions if reading/writing to S3

## Security & Performance

### Security Considerations

- ⚠️ Implement URL validation to prevent SSRF attacks
- Use allowlist for permitted domains/origins
- Configure CORS headers appropriately
- Enable AWS WAF for API Gateway
- Use CloudFront with signed URLs for sensitive content
- Monitor CloudWatch Logs for suspicious activity
- Validate image dimensions and file sizes to prevent DoS

### Performance Tips

- Warm up Lambda with scheduled CloudWatch Events
- Use Provisioned Concurrency for consistent performance
- Implement caching layer (CloudFront, S3) for frequently accessed images
- Monitor X-Ray traces to identify bottlenecks
- Consider Lambda@Edge for CDN integration

### Cost Optimization

- Use HTTP API instead of REST API (cheaper)
- Set appropriate memory allocation (start with 512 MB)
- Configure shorter timeout for typical use cases
- Enable CloudWatch Logs Insights for monitoring
- Use S3 lifecycle policies if caching to S3

## Important Notes

### Critical Build Requirements

1. **Package Name:** `src/main.go` must use `package main` (not `package imgop`)
2. **CGO Required:** Build must use `CGO_ENABLED=1` for libvips
3. **Architecture:** Must build for `GOOS=linux GOARCH=amd64`
4. **Lambda Layer:** Required for runtime libraries
5. **Environment Variable:** Lambda needs `LD_LIBRARY_PATH=/opt/lib`

### Lambda Layer Structure

The libvips layer must have this structure:
```
lib/
  libvips.so.42
  libvips.so.42.19.2
  libvips-cpp.so.42
  (and other .so files)
bin/
  vips
  vipsheader
  vipsthumbnail
```

- Lambda automatically mounts layers at `/opt`
- `LD_LIBRARY_PATH=/opt/lib` tells the dynamic linker to look there
- Make sure bootstrap binary is executable: `chmod +x build/bootstrap`

### File Organization Principles

- **Source code** → `src/`
- **Build outputs** → `build/`
- **Docker files** → `deployment-scripts/docker/`
- **Test scripts** → `deployment-scripts/test/`
- **Documentation** → Root directory (README.md)
- **Deployment logic** → `deployment-scripts/`

This organization keeps the root clean and groups related functionality together.

---

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]

## Support

For issues and questions:
- Check the [Troubleshooting](#troubleshooting) section
- Review CloudWatch Logs for error details
- Verify build artifacts with `make verify`
