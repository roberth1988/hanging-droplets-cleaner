export VERSION ?= $(shell git describe 2>/dev/null | sed -e 's/^v//g' || echo "dev")
export REVISION ?= $(shell git rev-parse --short HEAD || echo "unknown")
export BRANCH ?= $(shell git show-ref | grep "$(REVISION)" | grep -v HEAD | awk '{print $$2}' | sed 's|refs/remotes/origin/||' | sed 's|refs/heads/||' | sort | head -n 1)
export BUILT ?= $(shell date +%Y-%m-%dT%H:%M:%S%:z)

PROJECT_PACKAGES ?= $(shell go list ./... | grep -v vendor)
VERSION_PACKAGE ?= $(shell go list ./version)

export CGO_ENABLED := 0
GO_LDFLAGS := -X $(VERSION_PACKAGE).VERSION=$(VERSION) \
              -X $(VERSION_PACKAGE).REVISION=$(REVISION) \
              -X $(VERSION_PACKAGE).BRANCH=$(BRANCH) \
              -X $(VERSION_PACKAGE).BUILT=$(BUILT) \
              -s -w

version: FORCE
	@echo Current version: $(VERSION)
	@echo Current revision: $(REVISION)
	@echo Current branch: $(BRANCH)

static_code_analysis: fmt vet lint complexity

deps:
	# Installing required dependencies
	@go get github.com/golang/dep/cmd/dep
	@go get github.com/golang/lint/golint
	@go get golang.org/x/tools/cmd/cover
	@go get github.com/fzipp/gocyclo
	@go install cmd/vet
	@dep ensure

fmt:
	# Checking project code formatting...
	@go fmt $(PROJECT_PACKAGES) | awk '{if (NF > 0) {if (NR == 1) print "Please run go fmt for:"; print "- "$$1}} END {if (NF > 0) {if (NR > 0) exit 1}}'

vet:
	# Checking for suspicious constructs...
	@go vet $(PROJECT_PACKAGES)

lint:
	# Checking project code style...
	@golint $(PROJECT_PACKAGES) | ( ! grep -v -e "be unexported" -e "don't use an underscore in package name" -e "ALL_CAPS" )

complexity:
	# Checking code complexity
	@gocyclo -over 5 $(shell find . -name '*.go' | grep -v -e "/vendor/")

test:
	# Running unit tests
	@go test $(PROJECT_PACKAGES)

compile:
	# Compile binary
	@go build -ldflags "$(GO_LDFLAGS)"
	# Test run
	@./hanging-droplets-cleaner --version

release_image:
	# Release image
	@./scripts/release_image

release_ci_image:
	# Release CI image
	@./scripts/release_ci_image

FORCE:
