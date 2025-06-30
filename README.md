# k8s-controller

[![CI](https://github.com/k0rvih/k8s-controller/actions/workflows/ci.yml/badge.svg)](https://github.com/k0rvih/k8s-controller/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/k0rvih/k8s-controller)](https://goreportcard.com/report/github.com/k0rvih/k8s-controller)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Kubernetes controller and CLI tool built with Go, featuring structured logging, HTTP server capabilities, and cloud-native deployment patterns.

## ✨ Features

- **🚀 CLI Interface**: Built with [Cobra](https://github.com/spf13/cobra) for intuitive command-line operations
- **🌐 HTTP Server**: High-performance FastHTTP server with configurable ports
- **📊 Structured Logging**: Advanced logging with [zerolog](https://github.com/rs/zerolog) supporting multiple levels and formats
- **🐳 Container Ready**: Distroless Docker images for security and minimal attack surface
- **☸️ Kubernetes Native**: Helm charts for easy Kubernetes deployment
- **🔧 CI/CD Pipeline**: Automated testing, building, and deployment with GitHub Actions
- **🛡️ Security Scanning**: Integrated Trivy security scanning for container images

## 📁 Project Structure

```
k8s-controller/
├── cmd/                          # CLI commands
│   ├── root.go                   # Root command with logging configuration
│   ├── server.go                 # HTTP server command
│   └── server_test.go            # Server command tests
├── charts/                       # Helm charts
│   └── app/                      # Application Helm chart
│       ├── Chart.yaml            # Chart metadata
│       ├── values.yaml           # Default values
│       ├── README.md             # Chart documentation
│       └── templates/            # Kubernetes manifests
│           ├── deployment.yaml   # Application deployment
│           ├── service.yaml      # Service definition
│           └── _helpers.tpl      # Template helpers
├── .github/workflows/            # GitHub Actions
│   └── ci.yml                    # CI/CD pipeline
├── main.go                       # Application entry point
├── Dockerfile                    # Container image definition
├── Makefile                      # Build automation
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
└── README.md                     # This file
```

## 🚀 Installation

### Prerequisites

- Go 1.24.2 or later
- Docker (for containerization)
- Kubernetes cluster (for deployment)
- Helm 3.x (for Kubernetes deployment)

### From Source

```bash
# Clone the repository
git clone https://github.com/k0rvih/k8s-controller.git
cd k8s-controller

# Build the application
make build

# Or build with custom version
VERSION=v1.0.0 make build
```

### Using Go Install

```bash
go install github.com/k0rvih/k8s-controller@latest
```

### Using Docker

```bash
# Pull the latest image
docker pull ghcr.io/k0rvih/k8s-controller/app:latest

# Run the container
docker run --rm -p 8080:8080 ghcr.io/k0rvih/k8s-controller/app:latest server
```

## 📖 Usage

### CLI Commands

The application provides several commands for different operations:

#### Root Command
```bash
k8s-controller --log-level info
```

**Available log levels:** `trace`, `debug`, `info`, `warn`, `error`

#### HTTP Server
```bash
# Start server on default port (8080)
k8s-controller server

# Start server on custom port
k8s-controller server --port 3000

# Start with debug logging
k8s-controller server --log-level debug --port 8080
```

### Docker Usage

```bash
# Run the application
docker run --rm ghcr.io/k0rvih/k8s-controller/app:latest --help

# Run the server
docker run --rm -p 8080:8080 ghcr.io/k0rvih/k8s-controller/app:latest server

# Run with custom configuration
docker run --rm -p 3000:3000 ghcr.io/k0rvih/k8s-controller/app:latest server --port 3000 --log-level debug
```

### Kubernetes Deployment

Deploy using Helm:

```bash
# Add your repository (if published to a Helm repo)
helm repo add k8s-controller https://your-helm-repo.com

# Install with default values
helm install my-app ./charts/app

# Install with custom values
helm install my-app ./charts/app \
  --set image.tag=v1.0.0 \
  --set image.pullPolicy=Always

# Upgrade deployment
helm upgrade my-app ./charts/app --set image.tag=v1.1.0
```

## 🛠️ Development

### Building

```bash
# Build for current platform
make build

# Build for specific platform
GOOS=linux GOARCH=amd64 make build

# Build Docker image
make docker-build

# Clean build artifacts
make clean
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Local Development

```bash
# Run without building
make run

# Run server directly
go run main.go server --port 8080 --log-level debug
```

## 🔧 Configuration

### Environment Variables

The application can be configured using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_LEVEL` | Logging level (trace, debug, info, warn, error) | `info` |
| `PORT` | Server port | `8080` |

### Logging Configuration

The application supports multiple logging formats:

- **Trace Level**: Includes caller information and detailed console output
- **Debug Level**: Console formatted output with timestamps
- **Info+ Levels**: JSON structured logging to stderr

## 🐳 Container Images

Images are automatically built and published to GitHub Container Registry:

- **Latest**: `ghcr.io/k0rvih/k8s-controller/app:latest`
- **Tagged**: `ghcr.io/k0rvih/k8s-controller/app:v1.0.0`
- **Branch**: `ghcr.io/k0rvih/k8s-controller/app:0.1.0-abcd1234`

Images are built using:
- **Multi-stage builds** for optimal size
- **Distroless base images** for security
- **Non-root user** for enhanced security
- **Security scanning** with Trivy

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Quality

- Ensure all tests pass: `make test`
- Follow Go best practices and formatting
- Add tests for new functionality
- Update documentation as needed

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- [Go](https://golang.org/) - Programming language
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Zerolog](https://github.com/rs/zerolog) - Structured logging
- [FastHTTP](https://github.com/valyala/fasthttp) - HTTP framework
- [Helm](https://helm.sh/) - Kubernetes package manager

---

**Made with ❤️ by [k0rvih](https://github.com/k0rvih)**