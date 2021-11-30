GOCMD=GO111MODULE=on go
GOFMT=$(GOCMD) fmt

# Build-time GIT variables
ifeq ($(GIT_COMMIT),)
GIT_COMMIT:=$(shell git rev-parse HEAD)
endif

ifeq ($(GIT_VERSION),)
GIT_VERSION:=$(shell git describe --tags --dirty)
endif

LDFLAGS := "-s -w -X github.com/codeperfio/pprof-exporter/cmd.Version=$(GIT_VERSION) -X github.com/codeperfio/pprof-exporter/cmd.GitCommit=$(GIT_COMMIT)"
export GO111MODULE=on
SOURCE_DIRS = cmd main.go

.PHONY: all
all: dist hash

.PHONY: gofmt
gofmt:
	@test -z $(shell gofmt -l -s $(SOURCE_DIRS) ./ | tee /dev/stderr) || (echo "[WARN] Fix formatting issues with 'make fmt'" && exit 1)

test:
	$(GOCMD) test -count=1 -v ./cmd/.

build:
	$(GOCMD) build .

fmt:
	$(GOFMT) ./*.go
	$(GOFMT) ./cmd/*.go

lint:
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint
	golangci-lint run

.PHONY: dist
dist:
	mkdir -p bin/
	rm -rf bin/pprof-exporter*
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o bin/pprof-exporter
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o bin/pprof-exporter-darwin
	GOARM=6 GOARCH=arm CGO_ENABLED=0 GOOS=linux go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o bin/pprof-exporter-armhf
	GOARCH=arm64 CGO_ENABLED=0 GOOS=linux go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o bin/pprof-exporter-arm64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags $(LDFLAGS) -installsuffix cgo -o bin/pprof-exporter.exe

.PHONY: hash
hash:
	rm -rf bin/*.sha256 && ./scripts/hash.sh