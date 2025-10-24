# Image Operations Lambda (imgop)

A serverless image optimization service running on AWS Lambda that converts and optimizes images to WebP format with customizable dimensions and quality settings.

## Quick Start

```bash
# 1. Build for Lambda
make deploy

# 2. Deploy to AWS
- Upload build/libvips.zip as Lambda Layer
- Upload build/app.zip as Lambda Function
```

## Requirements

- Docker (for building in Amazon Linux 2023 environment)
- AWS CLI configured with credentials
- Go 1.21+ (for local development only)

## Lambda Setup

For setting up lambda, you can use this configuration:
- Runtime: `Amazon Linux 2023`
- Handler: `bootstrap`
- Architecture: `x86_64`
- Configure -> Environment:
  - `ALLOWED_ORIGINS=yoursite.com,static.yoursite.com`
  - `LD_LIBRARY_PATH=/opt/bin:/opt/lib:/opt/lib64`

For hardware configuration, you can use the default minimum configuration:
- Memory - 128MB
- Ephemeral storage: 512MB

## Building

### Deploy Build (Production)

```bash
make deploy
```

This builds everything in an Amazon Linux 2023 Docker container to ensure compatibility with AWS Lambda:
- `build/bootstrap` - Lambda function binary (GLIBC-compatible)
- `build/app.zip` - Function deployment package
- `build/libvips.zip` - Lambda layer with libvips 8.17.2 and runtime libraries

### Dev Build (Local Testing)

```bash
make dev
```

Builds the bootstrap binary locally for quick testing. **Note:** May not work on Lambda due to GLIBC version differences.

## AWS Lambda Deployment - Script

### Step 1: Create Lambda Layer

Upload the libvips layer:

```bash
aws lambda publish-layer-version \
    --layer-name libvips-runtime \
    --description "libvips 8.17.2 with runtime libraries" \
    --zip-file fileb://build/libvips.zip \
    --compatible-runtimes provided.al2023 \
    --compatible-architectures x86_64
```

Save the Layer ARN from the output.

### Step 2: Create Lambda Function

```bash
aws lambda create-function \
    --function-name imgop \
    --runtime provided.al2023 \
    --role arn:aws:iam::YOUR_ACCOUNT:role/YOUR_LAMBDA_ROLE \
    --handler bootstrap \
    --zip-file fileb://build/app.zip \
    --timeout 30 \
    --memory-size 128 \
    --architectures x86_64 \
    --layers arn:aws:lambda:REGION:ACCOUNT:layer:libvips-runtime:VERSION
```

### Step 3: Configure Environment

**Critical:** Set the library path so Lambda can find libvips:

```bash
aws lambda update-function-configuration \
    --function-name imgop \
    --environment "Variables={LD_LIBRARY_PATH=/opt/bin:/opt/lib:/opt/lib64}"
```

### Step 4: Set Up API Gateway

```bash
# Create HTTP API
aws apigatewayv2 create-api \
    --name imgop-api \
    --protocol-type HTTP \
    --target arn:aws:lambda:REGION:ACCOUNT:function:imgop

# Grant permission
aws lambda add-permission \
    --function-name imgop \
    --statement-id apigateway-invoke \
    --action lambda:InvokeFunction \
    --principal apigatewayv2.amazonaws.com
```

### Step 5: Test

```bash
curl "https://YOUR_API_URL/?url=https://via.placeholder.com/800.jpg&w=400&q=85" \
    --output optimized.webp
```

## API Parameters

| Parameter | Required | Description | Default |
|-----------|----------|-------------|---------|
| `url` | Yes | URL of image to optimize | - |
| `w` | No | Target width in pixels | Original |
| `h` | No | Target height in pixels | Original |
| `q` | No | Quality (1-100) | 80 |

## Updating

When you make code changes:

```bash
# Rebuild
make deploy

# Update function
aws lambda update-function-code \
    --function-name imgop \
    --zip-file fileb://build/app.zip
```

## Configuration

### Lambda Function Settings

| Setting | Recommended Value | Notes |
|---------|-------------------|-------|
| Runtime | `provided.al2023` | Custom runtime |
| Handler | `bootstrap` | Binary name |
| Memory | 512-1024 MB | Image processing needs memory |
| Timeout | 30 seconds | Adjust for large images |
| Architecture | `x86_64` | Must match build |

### Environment Variables

**Required:**
- `LD_LIBRARY_PATH` = `/opt/bin:/opt/lib:/opt/lib64`

This tells Lambda where to find libvips and its dependencies.

### IAM Permissions

Lambda execution role needs:
- `logs:CreateLogGroup`
- `logs:CreateLogStream`
- `logs:PutLogEvents`

## Troubleshooting

### Exit Status 127

**Problem:** "error while loading shared libraries"

**Solution:**
1. Verify layer is attached to function
2. Check `LD_LIBRARY_PATH=/opt/bin:/opt/lib:/opt/lib64` is set
3. Verify layer contains libraries: `unzip -l build/libvips.zip | grep .so`

### Segmentation Fault

**Problem:** "signal: segmentation fault"

**Solution:**
1. Ensure you used `make deploy` (not `make dev`) 
2. Increase memory to 512 MB minimum
3. Verify Runtime is `provided.al2023` (not `provided.al2`)

### Exec Format Error

**Problem:** "exec format error"

**Solution:**
- Ensure `src/main.go` uses `package main` (not `package imgop`)
- Rebuild with `make deploy`

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make deploy` | Build for Lambda (Docker-based, compatible build) |
| `make dev` | Build locally for testing |
| `make install` | Install/update Go dependencies |
| `make upgrade` | Upgrade Go dependencies |

## Project Structure

```
imgop/
├── src/
│   ├── main.go              # Lambda handler (package main)
│   ├── main_test.go
│   └── libs/
│       └── image-optimizer.go
├── deployment-scripts/
│   ├── build.sh             # Build orchestration
│   └── docker/
│       └── Dockerfile.build # Amazon Linux 2023 build environment
├── build/                    # Build outputs
│   ├── bootstrap            # Function binary
│   ├── app.zip              # Function package
│   └── libvips.zip          # Lambda layer
├── go.mod
├── makefile
└── README.md
```

## How It Works

1. API Gateway receives request with image URL and parameters
2. Lambda function downloads the source image
3. libvips processes the image (resize, optimize)
4. Converts to WebP format
5. Returns base64-encoded image
6. API Gateway serves the optimized image

## Security

- ⚠️ Implement URL validation to prevent SSRF attacks
- Use allowlist for permitted domains
- Enable AWS WAF on API Gateway
- Monitor CloudWatch Logs for suspicious activity
- Validate image dimensions to prevent DoS

## Performance

- Use CloudFront for caching
- Enable Provisioned Concurrency for consistent performance
- Monitor with X-Ray for bottlenecks
- Consider Lambda@Edge for CDN integration

## Notes

- Bootstrap binary is built in Amazon Linux 2023 for GLIBC compatibility
- Layer contains libvips 8.17.2 + all runtime dependencies
- Docker build ensures compatibility with Lambda environment

## License

MIT License
