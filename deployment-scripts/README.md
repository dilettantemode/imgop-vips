# Deployment Scripts

This directory contains all deployment, build, and testing infrastructure for the imgop Lambda function.

## Directory Structure

```
deployment-scripts/
├── docker/                      # Docker-related files
│   ├── Dockerfile.build-vips   # Builds libvips layer for Lambda
│   ├── Dockerfile.lambda-build # Lambda-compatible build environment (future use)
│   └── Dockerfile.test         # Local testing environment
│
├── test/                        # Testing and verification
│   ├── template.yaml           # AWS SAM template for local testing
│   ├── test-event.json         # Sample Lambda event
│   ├── test-sam.sh            # Test using AWS SAM CLI
│   ├── test-local.sh          # Test using Docker
│   └── verify-build.sh        # Verify build artifacts
│
├── ec2/                         # EC2 deployment scripts (if needed)
│   ├── bootstrap.sh
│   ├── build.sh
│   ├── deploy.sh
│   └── run.sh
│
├── build-libvips.sh            # Build libvips Lambda layer
├── build.sh                    # Build script
└── README.md                   # This file
```

## Usage

### Building

```bash
# From project root
make deploy              # Build Lambda function
make build-libvips      # Build libvips layer
```

### Testing

```bash
# From project root
make verify             # Verify build artifacts
make test-sam          # Test with AWS SAM CLI
make test-local        # Test with Docker
```

### Direct Script Usage

All scripts can also be called directly from the project root:

```bash
# Build libvips layer
./deployment-scripts/build-libvips.sh

# Verify build
./deployment-scripts/test/verify-build.sh

# Test with SAM
./deployment-scripts/test/test-sam.sh

# Test with Docker
./deployment-scripts/test/test-local.sh
```

## Docker Files

### Dockerfile.build-vips
Builds libvips 8.17.2 in an Alpine Linux environment compatible with AWS Lambda. Outputs a zip file containing:
- `bin/` - vips executables
- `lib/` - shared libraries (critical for runtime)

### Dockerfile.test
Creates a local Lambda testing environment using AWS Lambda runtime image. Used for testing the function before deployment.

### Dockerfile.lambda-build
(Future use) Builds the Go binary in a Lambda-compatible environment to ensure library compatibility.

## Test Files

### template.yaml
AWS SAM template that defines the Lambda function configuration for local testing.

### test-event.json
Sample API Gateway event for testing the image optimization function:
```json
{
  "queryStringParameters": {
    "url": "https://via.placeholder.com/1200x800.jpg",
    "w": "800",
    "h": "600",
    "q": "85"
  }
}
```

### Scripts

- **verify-build.sh**: Checks that build artifacts are correctly formatted for Lambda
- **test-sam.sh**: Uses AWS SAM CLI to test the function locally
- **test-local.sh**: Uses Docker to simulate Lambda environment

## Notes

- All test scripts automatically navigate to the project root, so they work from any directory
- Scripts assume `build/bootstrap` and `build/libvips.zip` exist
- Docker must be running for build-libvips and test-local scripts
- AWS SAM CLI must be installed for test-sam script

## See Also

- [Main README](../README.md) - Project overview and quick start
- [DEPLOY_GUIDE](../DEPLOY_GUIDE.md) - Complete deployment instructions
- [FIXES_APPLIED](../FIXES_APPLIED.md) - Technical details about fixes

