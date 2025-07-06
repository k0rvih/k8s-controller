# k8s-controller

[![CI](https://github.com/k0rvih/k8s-controller/actions/workflows/ci.yml/badge.svg)](https://github.com/k0rvih/k8s-controller/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/k0rvih/k8s-controller)](https://goreportcard.com/report/github.com/k0rvih/k8s-controller)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A comprehensive Kubernetes controller CLI tool built with Go that provides deployment management and real-time event monitoring capabilities.

## Features

- 🚀 **Kubernetes Resource Management**: Create, list, and delete deployments and pods
- 📊 **Real-time Informer**: Watch and log Kubernetes deployment events (add, update, delete)
- 🎯 **Advanced Controller**: Controller-runtime based deployment controller with detailed event logging
- 👑 **Leader Election**: High availability support with configurable leader election
- 📈 **Metrics Server**: Built-in Prometheus metrics endpoint for monitoring
- 🔐 **Flexible Authentication**: Support for kubeconfig file, environment variables, and in-cluster authentication
- 🌐 **HTTP Server**: Built-in FastHTTP server with optional informer integration
- 🧪 **Comprehensive Testing**: Unit tests with envtest for realistic Kubernetes API testing
- 📝 **Structured Logging**: Professional logging with zerolog

## Installation

```bash
# Clone the repository
git clone https://github.com/k0rvih/k8s-controller.git
cd k8s-controller

# Build the application
go build -o k8s-controller .
```

## Usage

### 1. List Kubernetes Resources

```bash
# List deployments in default namespace
./k8s-controller list deployments

# List pods in specific namespace
./k8s-controller list pods --namespace production

# Use custom kubeconfig
./k8s-controller list deployments --kubeconfig ~/.kube/staging-config
```

### 2. Create Resources

```bash
# Create nginx deployment with 3 replicas
./k8s-controller create deployment nginx-app nginx:latest --replicas 3

# Create a standalone pod
./k8s-controller create pod test-pod busybox:latest

# Create in specific namespace
./k8s-controller create deployment api-server node:16 --namespace production --replicas 5
```

### 3. Delete Resources

```bash
# Delete deployment
./k8s-controller delete deployment nginx-app

# Delete pod
./k8s-controller delete pod test-pod

# Delete from specific namespace
./k8s-controller delete deployment api-server --namespace production
```

### 4. HTTP Server with Advanced Controller and Informers

The server runs a FastHTTP server and automatically starts both:
- **Deployment Informer**: Traditional informer for basic deployment events
- **Advanced Controller**: Controller-runtime based controller with detailed deployment analysis
- **Leader Election**: Optional leader election for high availability deployments

#### Start HTTP Server

```bash
# Basic server with informer and controller using default kubeconfig
./k8s-controller server

# Custom port and kubeconfig
./k8s-controller server --port 8080 --kubeconfig ~/.kube/config

# Use environment variable for kubeconfig (PowerShell)
$env:KUBECONFIG = "C:\Users\user\.kube\config"
./k8s-controller server

# Use environment variable for kubeconfig (Bash)
export KUBECONFIG=/path/to/config
./k8s-controller server

# Use in-cluster authentication (when running in a Pod)
./k8s-controller server --in-cluster

# Enable leader election for high availability
./k8s-controller server --enable-leader-election --leader-election-namespace kube-system

# Custom metrics port
./k8s-controller server --metrics-port 9090
```

#### Server Endpoints

```bash
# Default endpoint
curl http://localhost:8080
# Response: Hello from FastHTTP!

# Get list of deployments
curl http://localhost:8080/deployments
# Response: ["nginx-app", "api-server"]

# Get controller metrics (Prometheus format)
curl http://localhost:8081/metrics
# Response: Prometheus metrics including controller performance data
```

Example responses:
```bash
# Default endpoint
Hello from FastHTTP!

# /deployments endpoint
["nginx-app", "api-server"]

# /pods endpoint
["nginx-app-7d4b8c9f8d-abc123", "api-server-5f6g7h8i9j-def456"]
```

## Configuration

### Authentication Methods

1. **Command Line Parameter** (highest priority):
   ```bash
   --kubeconfig ~/.kube/config
   ```

2. **Environment Variable** (fallback):
   ```bash
   # PowerShell
   $env:KUBECONFIG = "C:\Users\user\.kube\config"

   # Bash
   export KUBECONFIG=/path/to/config
   ```

3. **In-Cluster Authentication** (for Pods):
   ```bash
   --in-cluster
   ```

### Command Line Options

#### Global Flags
- `--log-level`: Set logging level (trace, debug, info, warn, error)

#### Server Command
- `--port`: HTTP server port (default: 8080)
- `--kubeconfig`: Path to kubeconfig file (defaults to KUBECONFIG environment variable if not specified)
- `--in-cluster`: Use in-cluster authentication
- `--enable-leader-election`: Enable leader election for controller manager (default: true)
- `--leader-election-namespace`: Namespace for leader election (default: default)
- `--metrics-port`: Port for controller manager metrics (default: 8081)

Note: Both deployment and pod informers are always enabled and monitor the "default" namespace only.

#### Resource Commands
- `--namespace, -n`: Kubernetes namespace
- `--kubeconfig, -k`: Path to kubeconfig file
- `--replicas, -r`: Number of replicas (for deployments)

## Event Logging

The server provides two levels of deployment monitoring:

### 1. Basic Informer Events
Traditional informer events for deployment lifecycle:
```json
{"level":"info","time":"2025-01-01T20:30:15Z","message":"Deployment added: nginx-app"}
{"level":"info","time":"2025-01-01T20:31:20Z","message":"Deployment updated: nginx-app"}
{"level":"info","time":"2025-01-01T20:32:30Z","message":"Deployment deleted: nginx-app"}
```

### 2. Advanced Controller Events
Detailed controller-runtime based events with comprehensive deployment analysis:

#### Deployment Status Monitoring
```json
{
  "level":"info",
  "time":"2025-01-01T20:30:15Z",
  "namespace":"default",
  "name":"nginx-app",
  "desired_replicas":3,
  "current_replicas":3,
  "ready_replicas":3,
  "available_replicas":3,
  "updated_replicas":3,
  "message":"Deployment status"
}
```

#### Scaling Events
```json
{
  "level":"info",
  "time":"2025-01-01T20:31:20Z",
  "namespace":"default",
  "name":"nginx-app",
  "from_replicas":3,
  "to_replicas":5,
  "message":"🔄 Deployment scaling UP detected"
}
```

#### Deployment Conditions
```json
{
  "level":"info",
  "time":"2025-01-01T20:30:25Z",
  "namespace":"default",
  "name":"nginx-app",
  "condition_type":"Available",
  "status":"True",
  "reason":"MinimumReplicasAvailable",
  "message":"Deployment has minimum availability.",
  "last_transition":"2025-01-01T20:30:20Z",
  "message":"📋 Deployment condition"
}
```

#### Resource Usage Monitoring
```json
{
  "level":"info",
  "time":"2025-01-01T20:30:30Z",
  "namespace":"default",
  "name":"nginx-app",
  "container_name":"nginx",
  "requests":{"cpu":"100m","memory":"128Mi"},
  "limits":{"cpu":"500m","memory":"512Mi"},
  "message":"📊 Container resources"
}
```

### 3. Pod Events
```json
{"level":"info","time":"2025-01-01T20:30:20Z","pod":"nginx-app-7d4b8c9f8d-abc123","namespace":"default","phase":"Pending","message":"Pod added"}
{"level":"info","time":"2025-01-01T20:30:25Z","pod":"nginx-app-7d4b8c9f8d-abc123","namespace":"default","old_phase":"Pending","new_phase":"Running","message":"Pod phase updated"}
{"level":"info","time":"2025-01-01T20:32:35Z","pod":"nginx-app-7d4b8c9f8d-abc123","namespace":"default","message":"Pod deleted"}
```

### 4. Leader Election Events
```json
{"level":"info","time":"2025-01-01T20:30:10Z","message":"Starting controller-runtime manager..."}
{"level":"info","time":"2025-01-01T20:30:11Z","message":"Leader election enabled, waiting to acquire lease..."}
{"level":"info","time":"2025-01-01T20:30:12Z","message":"Successfully acquired leader lease"}
```

## Development

### Project Structure

```
k8s-controller/
├── charts/                    # Helm charts for deployment
│   └── app/
│       ├── Chart.yaml         # Helm chart metadata
│       ├── values.yaml        # Default values
│       ├── README.md          # Chart documentation
│       └── templates/         # Kubernetes manifest templates
│           ├── _helpers.tpl
│           ├── deployment.yaml
│           └── service.yaml
├── cmd/                       # CLI commands
│   ├── root.go                # Root command configuration
│   ├── root_test.go           # Root command tests
│   ├── list.go                # Resource listing commands
│   ├── list_test.go           # List command tests
│   ├── server.go              # HTTP server with informer integration
│   └── server_test.go         # Server command tests
├── pkg/
│   ├── ctrl/                  # Controller-runtime based controllers
│   │   ├── deployment_controller.go    # Advanced deployment controller
│   │   └── deployment_controller_test.go  # Controller tests
│   ├── informer/              # Kubernetes informer implementation
│   │   ├── informer.go        # Main informer logic
│   │   └── informer_test.go   # Informer tests
│   └── testutil/              # Testing utilities
│       ├── envtest.go         # envtest setup and helpers
│       └── envtest_test.go    # envtest tests
├── main.go                    # Application entry point
├── go.mod                     # Go module dependencies
├── Makefile                   # Build and test automation
├── Dockerfile                 # Container image definition
├── LICENSE                    # MIT license
└── README.md                  # Project documentation
```

### Running Tests

#### Unit Tests (without envtest)
```bash
# Test utility functions
go test ./pkg/informer -run TestGetDeploymentName -v

# Test configuration validation
go test ./pkg/informer -run TestInformerConfigValidation -v

# Test in-cluster config handling
go test ./pkg/informer -run TestInformerWithInClusterConfig -v
```

#### Integration Tests (with envtest)

For full integration testing, you need to install envtest binaries:

```bash
# Install envtest binaries
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
setup-envtest use 1.29.x!

# Run full test suite
go test ./pkg/informer -v
go test ./pkg/ctrl -v

# Run specific controller tests
go test ./pkg/ctrl -run TestDeploymentReconciler -v

# Run with inspection mode (uncomment sleep in test)
go test ./pkg/informer -run TestStartDeploymentInformer -v
```

When running envtest, a kubeconfig is written to `/tmp/envtest.kubeconfig` for debugging:

```bash
# Inspect the test cluster (while test is running)
kubectl --kubeconfig=/tmp/envtest.kubeconfig get all -A
kubectl --kubeconfig=/tmp/envtest.kubeconfig get deployments -n test-informer
```

### Building for Production

```bash
# Build optimized binary
go build -ldflags="-w -s" -o k8s-controller .

# Build with version info
VERSION=$(git describe --tags --always)
go build -ldflags="-w -s -X main.version=$VERSION" -o k8s-controller .
```

## Deployment Examples

### Local Development
```bash
# Start server with informer for local development
./k8s-controller server --kubeconfig ~/.kube/config
```

### Kubernetes Deployment with Leader Election
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-controller
spec:
  replicas: 2  # Multiple replicas for high availability
  selector:
    matchLabels:
      app: k8s-controller
  template:
    metadata:
      labels:
        app: k8s-controller
    spec:
      serviceAccountName: k8s-controller
      containers:
      - name: k8s-controller
        image: k8s-controller:latest
        args:
        - server
        - --in-cluster
        - --enable-leader-election
        - --leader-election-namespace=kube-system
        - --metrics-port=8081
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8081
          name: metrics
        env:
        - name: KUBECONFIG
          value: "/etc/kubeconfig/config"  # Optional: custom kubeconfig location
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: k8s-controller
spec:
  selector:
    app: k8s-controller
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 8081
    targetPort: 8081
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k8s-controller
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k8s-controller
rules:
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch", "create", "update", "delete"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch", "create", "delete"]
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: k8s-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: k8s-controller
subjects:
- kind: ServiceAccount
  name: k8s-controller
  namespace: default
```

## Troubleshooting

### Common Issues

1. **Kubeconfig not found**:
   ```bash
   # PowerShell - Set environment variable
   $env:KUBECONFIG = "C:\Users\user\.kube\config"
   ./k8s-controller server

   # Or use explicit path
   ./k8s-controller server --kubeconfig "C:\Users\user\.kube\config"
   ```

2. **Permission denied**:
   ```bash
   # Check RBAC permissions for deployments
   kubectl auth can-i list deployments
   kubectl auth can-i watch deployments

   # Check leader election permissions
   kubectl auth can-i create leases --namespace kube-system
   kubectl auth can-i update leases --namespace kube-system
   ```

3. **Leader election conflicts**:
   ```bash
   # Check current leader
   kubectl get leases -n kube-system k8s-controller-leader-election

   # Multiple instances should show leader election logs
   ./k8s-controller server --log-level debug
   ```

4. **Metrics not accessible**:
   ```bash
   # Check metrics endpoint
   curl http://localhost:8081/metrics

   # Verify metrics port is not blocked
   netstat -an | grep :8081
   ```

5. **Controller not reconciling**:
   ```bash
   # Check controller logs with debug level
   ./k8s-controller server --log-level debug

   # Verify deployment exists
   kubectl get deployments -n default

   # Check controller-runtime manager status
   curl http://localhost:8081/metrics | grep controller
   ```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

## References

- [Kubernetes client-go](https://github.com/kubernetes/client-go)
- [Cobra CLI](https://github.com/spf13/cobra)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Reference Implementation](https://github.com/den-vasyliev/k8s-controller-tutorial-ref/tree/feature/step7-informer)

## License

MIT License. See LICENSE for details.