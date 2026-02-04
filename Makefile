.PHONY: all build build-faiss test clean install run

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY_NAME = sagasu

all: build

build:
	CGO_ENABLED=1 go build -ldflags "-s -w -X main.version=$(VERSION)" \
		-o bin/$(BINARY_NAME) ./cmd/sagasu

build-faiss:
	CGO_ENABLED=1 go build -tags=faiss -ldflags "-s -w -X main.version=$(VERSION)" \
		-o bin/$(BINARY_NAME) ./cmd/sagasu

test:
	go test -v -race ./...

benchmark:
	go test -bench=. -benchmem ./test/benchmark/...

clean:
	rm -rf bin/

install: build
	cp bin/$(BINARY_NAME) /usr/local/bin/
	mkdir -p /usr/local/etc/sagasu
	cp config.yaml.example /usr/local/etc/sagasu/config.yaml
	mkdir -p /usr/local/var/sagasu/data/models /usr/local/var/sagasu/data/indices/bleve /usr/local/var/sagasu/data/indices/faiss /usr/local/var/sagasu/data/db

run: build
	./bin/$(BINARY_NAME) server --config config.yaml.example

release:
	@echo "Creating release $(VERSION)"
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo ""
	@echo "Calculate SHA256:"
	@echo "curl -sL https://github.com/hyperjump/sagasu/archive/refs/tags/$(VERSION).tar.gz | shasum -a 256"
	@echo ""
	@echo "Update Formula/sagasu.rb with new version and SHA256"

test-formula:
	brew install --build-from-source Formula/sagasu.rb
	brew test sagasu
	brew services start sagasu
	sleep 2
	curl http://localhost:8080/health
	brew services stop sagasu
	brew uninstall sagasu
