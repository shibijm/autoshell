build: build-windows-amd64 build-linux-amd64 build-linux-arm64
build-%:
	$(eval include .env)
	$(eval export CGO_ENABLED = 0)
	$(eval OSARCH = $(subst -, ,$*))
	$(eval export GOOS = $(word 1,$(OSARCH)))
	$(eval export GOARCH = $(word 2,$(OSARCH)))
	@echo Building $*
	@go build -ldflags "-s -w -X autoshell/utils.devicePassSeed=$(DEVICE_PASS_SEED)" -trimpath -o out/$(GOOS)-$(GOARCH)/
	cp LICENSE COPYRIGHT NOTICE README.md out/$(GOOS)-$(GOARCH)/

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.txt -v ./...
	go tool cover -html=coverage.txt
	rm -f coverage.txt
