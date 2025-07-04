APP = k8s-controller-tutorial
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_FLAGS = -v -o $(APP) -ldflags "-X=github.com/den-vasyliev/$(APP)/cmd.appVersion=$(VERSION)"

# Detect OS and set appropriate paths and commands
ifeq ($(OS),Windows_NT)
    SHELL := powershell.exe
    .SHELLFLAGS := -NoProfile -Command
    LOCALBIN ?= $(shell (Get-Location).Path)/bin
    ENVTEST ?= $(LOCALBIN)/setup-envtest.exe
    EXE_EXT = .exe
else
    LOCALBIN ?= $(shell pwd)/bin
    ENVTEST ?= $(LOCALBIN)/setup-envtest
    EXE_EXT =
endif

ENVTEST_VERSION ?= latest

.PHONY: all build test test-coverage run docker-build clean envtest

all: build

## Location to install dependencies to
$(LOCALBIN):
ifeq ($(OS),Windows_NT)
	@if (!(Test-Path "$(LOCALBIN)")) { New-Item -ItemType Directory -Force -Path "$(LOCALBIN)" }
else
	mkdir -p $(LOCALBIN)
endif

## Tool Binaries
ENVTEST ?= $(LOCALBIN)/setup-envtest$(EXE_EXT)

## Tool Versions
ENVTEST_VERSION ?= release-0.19

format:
	gofmt -s -w ./

lint:
	golint

envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) main.go

test: envtest
	go install gotest.tools/gotestsum@latest
ifeq ($(OS),Windows_NT)
	$$env:KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --bin-dir $(LOCALBIN) -p path)"; gotestsum --junitfile report.xml --format testname ./... $(TEST_ARGS)
else
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --bin-dir $(LOCALBIN) -p path)" gotestsum --junitfile report.xml --format testname ./... ${TEST_ARGS}
endif

test-coverage: envtest
	go install github.com/boumenot/gocover-cobertura@latest
ifeq ($(OS),Windows_NT)
	$$env:KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --bin-dir $(LOCALBIN) -p path)"; go test -coverprofile=coverage.out -covermode=count ./...
else
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --bin-dir $(LOCALBIN) -p path)" go test -coverprofile=coverage.out -covermode=count ./...
endif
	go tool cover -func=coverage.out
	gocover-cobertura < coverage.out > coverage.xml

run:
	go run main.go

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(APP):latest .

clean:
ifeq ($(OS),Windows_NT)
	@if (Test-Path "$(APP)$(EXE_EXT)") { Remove-Item -Force "$(APP)$(EXE_EXT)" }
else
	rm -f $(APP)
endif

purge-envtest:
ifeq ($(OS),Windows_NT)
	@powershell -Command "Get-Process etcd, kube-apiserver -ErrorAction SilentlyContinue | Stop-Process -Force"
else
	@pkill -f etcd || true
	@pkill -f kube-apiserver || true
endif

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
ifeq ($(OS),Windows_NT)
define go-install-tool
	@if (!(Test-Path "$(1)-$(3)")) { \
		$$package = "$(2)@$(3)"; \
		Write-Host "Downloading $$package"; \
		if (Test-Path "$(1)") { Remove-Item -Force "$(1)" }; \
		$$env:GOBIN = "$(LOCALBIN)"; \
		go install $$package; \
		Move-Item "$(1)" "$(1)-$(3)"; \
	}; \
	Copy-Item "$(1)-$(3)" "$(1)" -Force
endef
else
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
endif