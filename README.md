# Image Operations Lambda (imgop)

A serverless image optimization service running on AWS Lambda that converts and optimizes images to WebP format with customizable dimensions and quality settings.

## Requirements
- Go 1.21+ installed
- Docker (for building libvips layer)
- AWS CLI configured with appropriate credentials
- AWS Lambda runtime environment
- libvips 8.17.2 for Lambda

## Local Development Setup

### 1. Install Dependencies
```bash
make install
```

### 2. Build for Development (lambda-x86_64)
```bash
make dev
```
This builds the Lambda function for x86_64 architecture (linux/amd64) and outputs to `build/bootstrap`.

### 3. Test Locally

#### Option A: Using AWS SAM CLI
```bash
# Install SAM CLI if you haven't already
# https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html

# Invoke locally
sam local invoke -e event.json
```

#### Option B: Using AWS Lambda Runtime Interface Emulator (RIE)
```bash
# Run the built binary with RIE
docker run -p 9000:8080 -v $(pwd)/build:/var/task amazon/aws-lambda-runtime-interface-emulator ./bootstrap
```

## AWS Lambda Deployment Guide

### Step 1: Build libvips Layer for Lambda

The application requires libvips 8.17.2 to be available in the Lambda runtime environment. We package this as a Lambda layer.

#### Build the libvips layer:
```bash
make build-libvips
```

This command will:
1. Use Docker to build libvips 8.17.2 in an Alpine Linux environment (compatible with Lambda)
2. Compile libvips with required dependencies (libjpeg-turbo, libpng, libwebp, etc.)
3. Package everything into `libvips-8.17.2-lambda-x86_64.zip`

**Note:** This process takes several minutes. The resulting ZIP file (~5-10MB) contains all necessary libvips binaries and libraries.

#### Create Lambda Layer:

1. **Via AWS Console:**
   - Navigate to AWS Lambda → Layers
   - Click "Create layer"
   - Name: `libvips-8-17-2` (or your preferred name)
   - Upload `libvips-8.17.2-lambda-x86_64.zip`
   - Compatible runtimes: `Custom runtime on Amazon Linux 2`
   - Compatible architectures: `x86_64`
   - Click "Create"

2. **Via AWS CLI:**
```bash
aws lambda publish-layer-version \
    --layer-name libvips-8-17-2 \
    --description "libvips 8.17.2 for image processing" \
    --zip-file fileb://libvips-8.17.2-lambda-x86_64.zip \
    --compatible-runtimes provided.al2 \
    --compatible-architectures x86_64
```

Save the Layer ARN from the output (e.g., `arn:aws:lambda:us-east-1:123456789012:layer:libvips-8-17-2:1`)

### Step 2: Build the Application

Build the application for Lambda:
```bash
make deploy
```

This command will:
1. Compile the Go application for Linux/AMD64
2. Create `build/bootstrap` executable
3. Package it into `build/app.zip`

The resulting `build/app.zip` contains your Lambda function code.

### Step 3: Create Lambda Function

#### Via AWS Console:

1. Navigate to AWS Lambda → Functions
2. Click "Create function"
3. Choose "Author from scratch"
4. Configuration:
   - **Function name:** `imgop` (or your preferred name)
   - **Runtime:** `Amazon Linux 2`
   - **Architecture:** `x86_64`
   - **Execution role:** Create new role or use existing with basic Lambda permissions
5. Click "Create function"

6. Upload function code:
   - In the "Code" tab, click "Upload from" → ".zip file"
   - Upload `build/app.zip`
   - Click "Save"

7. Add the libvips layer:
   - Scroll to "Layers" section
   - Click "Add a layer"
   - Choose "Custom layers"
   - Select your `libvips-8-17-2` layer
   - Select the version you created
   - Click "Add"

8. Configure function settings:
   - **Handler:** `bootstrap` (this is the default for custom runtimes)
   - **Runtime:** `Custom runtime on Amazon Linux 2` (provided.al2)
   - **Memory:** 512 MB (minimum recommended, adjust based on image sizes)
   - **Timeout:** 30 seconds (adjust based on your needs)
   - **Ephemeral storage:** 512 MB (default is fine)

9. Environment variables (optional):
   - `LD_LIBRARY_PATH`: `/opt/lib` (if libvips libraries are in this path)
   - `PATH`: `/opt/bin:$PATH` (to include vips binaries)

#### Via AWS CLI:

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

# Add the libvips layer (replace with your Layer ARN)
aws lambda update-function-configuration \
    --function-name imgop \
    --layers arn:aws:lambda:REGION:ACCOUNT_ID:layer:libvips-8-17-2:1
```

### Step 4: Set Up API Gateway

To make your Lambda function accessible via HTTP/HTTPS:

#### Via AWS Console:

1. Navigate to API Gateway → Create API
2. Choose "HTTP API" (simpler and cheaper) or "REST API" (more features)
3. Click "Build"
4. Configuration:
   - **API name:** `imgop-api`
   - **Integration:** AWS Lambda
   - **Lambda function:** Select your `imgop` function
   - **API Gateway version:** 2.0 (for HTTP API)
5. Configure routes:
   - **Method:** `GET`
   - **Resource path:** `/` or `/optimize`
6. Configure stages:
   - **Stage name:** `prod` or `$default`
7. Create and deploy

8. Note the Invoke URL (e.g., `https://abc123.execute-api.us-east-1.amazonaws.com/`)

#### Via AWS CLI:

```bash
# Create HTTP API
aws apigatewayv2 create-api \
    --name imgop-api \
    --protocol-type HTTP \
    --target arn:aws:lambda:REGION:ACCOUNT_ID:function:imgop

# Grant API Gateway permission to invoke Lambda
aws lambda add-permission \
    --function-name imgop \
    --statement-id apigateway-invoke \
    --action lambda:InvokeFunction \
    --principal apigatewayv2.amazonaws.com
```

### Step 5: Test Your Deployment

Test the deployed function:

```bash
# Test with curl
curl "https://YOUR_API_GATEWAY_URL/?url=https://example.com/image.jpg&w=800&q=85"
```

The API accepts these query parameters:
- `url` (required): URL of the image to optimize
- `w` (optional): Target width in pixels
- `h` (optional): Target height in pixels  
- `q` (optional): Quality (1-100, default: 80)

Example:
```bash
curl "https://abc123.execute-api.us-east-1.amazonaws.com/?url=https://example.com/sample.jpg&w=1200&h=800&q=90" \
    --output optimized.webp
```

### Step 6: Update Existing Deployment

When you make code changes, redeploy:

```bash
# Build new version
make deploy

# Update Lambda function code via AWS CLI
aws lambda update-function-code \
    --function-name imgop \
    --zip-file fileb://build/app.zip

# Or via AWS Console: Upload new app.zip in the Code tab
```

## Lambda Function Configuration

### Recommended Settings

| Setting | Value | Notes |
|---------|-------|-------|
| Memory | 512 MB - 1024 MB | Adjust based on image sizes |
| Timeout | 30 seconds | Adjust for larger images |
| Ephemeral storage | 512 MB | Default is sufficient |
| Architecture | x86_64 | Must match libvips build |
| Runtime | provided.al2 | Custom Go runtime |

### Required IAM Permissions

Your Lambda execution role needs:
- `logs:CreateLogGroup`
- `logs:CreateLogStream`
- `logs:PutLogEvents`
- (Optional) S3 permissions if reading/writing to S3

### Cost Optimization

- Use HTTP API instead of REST API (cheaper)
- Set appropriate memory allocation (start with 512 MB)
- Configure shorter timeout for typical use cases
- Consider Lambda@Edge for CDN integration
- Enable CloudWatch Logs Insights for monitoring

## Troubleshooting

### libvips not found
**Issue:** Lambda function fails with "libvips not found" or similar errors

**Solution:**
- Verify the libvips layer is attached to your function
- Check that the layer was built for the correct architecture (x86_64)
- Ensure environment variables are set correctly

### Image optimization fails
**Issue:** Function returns 400 error

**Solution:**
- Verify the source URL is accessible from Lambda
- Check image format is supported (JPEG, PNG, WebP, GIF)
- Ensure sufficient memory allocation
- Check CloudWatch Logs for detailed error messages

### Timeout errors
**Issue:** Function times out

**Solution:**
- Increase timeout setting in Lambda configuration
- Increase memory allocation (more memory = faster CPU)
- Check if source image is too large

## Makefile Commands

- `make install` - Install/update Go dependencies
- `make dev` - Build for local development (lambda-x86_64)
- `make deploy` - Build application and package for Lambda deployment
- `make build-libvips` - Build libvips layer for Lambda (requires Docker)
- `make upgrade` - Upgrade Go module dependencies

## Architecture

This Lambda function:
1. Receives image URL and optimization parameters via API Gateway
2. Downloads the image from the provided URL
3. Uses libvips to resize/optimize the image
4. Converts to WebP format
5. Returns the optimized image as base64-encoded response
6. API Gateway decodes and serves the binary image

## Security Considerations

- Implement URL validation to prevent SSRF attacks
- Use allowlist for permitted domains/origins
- Configure CORS headers appropriately
- Enable AWS WAF for API Gateway
- Use CloudFront with signed URLs for sensitive content
- Monitor CloudWatch Logs for suspicious activity

## Performance Tips

- Warm up Lambda with scheduled CloudWatch Events
- Use Provisioned Concurrency for consistent performance
- Implement caching layer (CloudFront, S3) for frequently accessed images
- Monitor X-Ray traces to identify bottlenecks
