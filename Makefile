BINARY_NAME ?= namesprout
DIST_DIR ?= dist
PACKAGE ?= ./cmd/namesprout

PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
LINUX_PLATFORMS ?= linux/amd64 linux/arm64
MAC_PLATFORMS ?= darwin/amd64 darwin/arm64
WINDOWS_PLATFORMS ?= windows/amd64 windows/arm64

.PHONY: build build-all build-linux build-mac build-windows build-platforms clean

build:
	@mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/$(BINARY_NAME) $(PACKAGE)

build-all:
	$(MAKE) build-platforms PLATFORMS="$(PLATFORMS)"

build-linux:
	$(MAKE) build-platforms PLATFORMS="$(LINUX_PLATFORMS)"

build-mac:
	$(MAKE) build-platforms PLATFORMS="$(MAC_PLATFORMS)"

build-windows:
	$(MAKE) build-platforms PLATFORMS="$(WINDOWS_PLATFORMS)"

build-platforms:
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output="$(DIST_DIR)/$(BINARY_NAME)-$${os}-$${arch}"; \
		if [ "$$os" = "windows" ]; then \
			output="$$output.exe"; \
		fi; \
		echo "==> Building $$output"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -o "$$output" $(PACKAGE) || exit $$?; \
	done

clean:
	rm -rf $(DIST_DIR)
