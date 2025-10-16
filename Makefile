BINARY_NAME ?= namesprout
PACKAGE ?= ./cmd/namesprout

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
OUTPUT := $(BINARY_NAME)

ifeq ($(GOOS),windows)
OUTPUT := $(OUTPUT).exe
endif

.PHONY: all clean

all:
	@echo "==> Building $(OUTPUT) for $(GOOS)/$(GOARCH)"
	go build -o $(OUTPUT) $(PACKAGE)

clean:
	rm -f $(OUTPUT)
