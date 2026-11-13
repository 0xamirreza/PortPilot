APP := portpilot
PKG := ./cmd/portpilot
BIN := bin/$(APP)
DIST := dist

.PHONY: build test lint release clean doctor deps install appimage

deps:
	@sh scripts/ensure-gui-deps.sh

build: deps
	@mkdir -p bin
	PKG_CONFIG_PATH="$(CURDIR)/build/third_party/pkgconfig$${PKG_CONFIG_PATH:+:$$PKG_CONFIG_PATH}" \
		go build -trimpath -ldflags="-s -w" -o $(BIN) $(PKG)

appimage: deps
	@sh scripts/build-appimage.sh

test: deps
	PKG_CONFIG_PATH="$(CURDIR)/build/third_party/pkgconfig$${PKG_CONFIG_PATH:+:$$PKG_CONFIG_PATH}" \
		go test ./...

lint: deps
	PKG_CONFIG_PATH="$(CURDIR)/build/third_party/pkgconfig$${PKG_CONFIG_PATH:+:$$PKG_CONFIG_PATH}" \
		go vet ./...
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

release: deps
	@mkdir -p $(DIST)
	PKG_CONFIG_PATH="$(CURDIR)/build/third_party/pkgconfig$${PKG_CONFIG_PATH:+:$$PKG_CONFIG_PATH}" \
		GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/portpilot-linux-amd64 $(PKG)
	@echo "amd64 build written"
	@if PKG_CONFIG_PATH="$(CURDIR)/build/third_party/pkgconfig$${PKG_CONFIG_PATH:+:$$PKG_CONFIG_PATH}" \
		GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(DIST)/portpilot-linux-arm64 $(PKG); then \
		echo "arm64 build written"; \
	else \
		echo "arm64 build failed; checksums cover only the binaries that exist" >&2; \
		rm -f $(DIST)/portpilot-linux-arm64; \
	fi
	cd $(DIST) && sha256sum portpilot-linux-* > SHA256SUMS

doctor: build
	./$(BIN) doctor

install: build
	@set -e; prefix="$${HOME}/.local"; \
	mkdir -p "$$prefix/bin"; \
	cp $(BIN) "$$prefix/bin/portpilot"; \
	"$$prefix/bin/portpilot" install-launcher; \
	echo "installed $$prefix/bin/portpilot"

clean:
	rm -rf bin dist build
