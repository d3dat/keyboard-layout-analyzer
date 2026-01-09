APP_NAME := kbda
VERSION ?= $(shell git describe --tags --always --dirty="-dev")
LDFLAGS := -s -w -extldflags "-static"

.PHONY: build clean release dist

build:
	CGO_ENABLED=0 go build -o $(APP_NAME) -ldflags="$(LDFLAGS)" ./cmd/$(APP_NAME)

clean:
	rm -f $(APP_NAME)
	rm -rf dist/
	rm -f $(APP_NAME)-*.zip $(APP_NAME)-*.tar.gz

dist:
	mkdir -p dist

release-linux-amd64: dist
	@echo "📦 Сборка linux/amd64"
	@mkdir -p dist/linux-amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/$(APP_NAME) -ldflags="$(LDFLAGS)" ./cmd/$(APP_NAME)
	cp -r configs/* README.md LICENSE dist/linux-amd64/
	tar -czf $(APP_NAME)-$(VERSION)-linux-amd64.tar.gz -C dist/linux-amd64 .

release-windows-amd64: dist
	@echo "📦 Сборка windows/amd64"
	@mkdir -p dist/windows-amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/$(APP_NAME).exe -ldflags="$(LDFLAGS)" ./cmd/$(APP_NAME)
	cp -r configs/* README.md LICENSE dist/windows-amd64/
	cd dist/windows-amd64 && zip -r ../../$(APP_NAME)-$(VERSION)-windows-amd64.zip .

release-darwin-amd64: dist
	@echo "📦 Сборка darwin/amd64"
	@mkdir -p dist/darwin-amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/$(APP_NAME) -ldflags="$(LDFLAGS)" ./cmd/$(APP_NAME)
	cp -r configs/* README.md LICENSE dist/darwin-amd64/
	tar -czf $(APP_NAME)-$(VERSION)-darwin-amd64.tar.gz -C dist/darwin-amd64 .

release-darwin-arm64: dist
	@echo "📦 Сборка darwin/arm64"
	@mkdir -p dist/darwin-arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/$(APP_NAME) -ldflags="$(LDFLAGS)" ./cmd/$(APP_NAME)
	cp -r configs/* README.md LICENSE dist/darwin-arm64/
	tar -czf $(APP_NAME)-$(VERSION)-darwin-arm64.tar.gz -C dist/darwin-arm64 .

release: release-linux-amd64 release-windows-amd64 release-darwin-amd64 release-darwin-arm64