APP_VERSION = 2.0.0

ifneq ($(wildcard ./.env),)
    include .env
    export
endif

build: build-windows-amd64 build-linux-amd64 build-linux-arm64
build-%:
	$(eval OSARCH := $(subst -, ,$*))
	$(eval GOOS := $(word 1,$(OSARCH)))
	$(eval GOARCH := $(word 2,$(OSARCH)))
	@echo Building $*
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-buildid= -s -w -X autoshell/cli.version=$(APP_VERSION) -X autoshell/config.devicePassSeed=$(DEVICE_PASS_SEED)" -trimpath -o out/$(GOOS)-$(GOARCH)/
	@cp NOTICE.md README.md LICENSE COPYRIGHT out/$(GOOS)-$(GOARCH)/

clean:
	rm -rf out

install-deps:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@6008b81b81c690c046ffc3fd5bce896da715d5fd

upgrade-deps:
	go get -u ./...
	go mod tidy

dev:
	nodemon --signal SIGKILL --ext go --exec "rm -f out/dev.exe && go build -o out/dev.exe"

lint:
	golangci-lint run --allow-parallel-runners

test:
	go test -count=1 -v ./...
