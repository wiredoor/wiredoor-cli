VERSION ?= 1.0.0
OUT_PATH ?= dist
BIN_PATH := bin
PKG_NAME := wiredoor
GO_MODULE := github.com/wiredoor/wiredoor-cli
ARCHS := amd64 arm64
WIN_ARCHS := amd64 arm64 386
MACOS_TMP := $(OUT_PATH)/macos-tmp
COMPLETIONS_DIR := completions
MAN_DIR := man
MAN_GZ_DIR := man-gz

build-pkgs: build-artifacts build-docs build-binaries build-deb build-rpm build-apk build-pacman build-windows build-macos clean

build-artifacts:
	chmod +x ./gen-winres.sh
	export VERSION=$(VERSION)
	./gen-winres.sh

build-docs:
	@rm -rf "$(COMPLETIONS_DIR)" "$(MAN_DIR)" "$(MAN_GZ_DIR)"
	@go run ./internal/tools/gencompletions --out "$(COMPLETIONS_DIR)"
	@go run ./internal/tools/genman --out "$(MAN_DIR)"
	@mkdir -p "$(MAN_GZ_DIR)"
	@find "$(MAN_DIR)" -name '*.1' -type f -exec sh -c 'gzip -9 -c "$$1" > "$(MAN_GZ_DIR)/$$(basename "$$1").gz"' _ {} \;

	@mkdir -p "usr/share/bash-completion/completions"
	@mkdir -p "usr/share/zsh/site-functions"
	@mkdir -p "usr/share/fish/vendor_completions.d"
	@mkdir -p "usr/share/man/man1"
	@cp "$(COMPLETIONS_DIR)/wiredoor.bash" "usr/share/bash-completion/completions/$(PKG_NAME)"
	@cp "$(COMPLETIONS_DIR)/_wiredoor" "usr/share/zsh/site-functions/_$(PKG_NAME)"
	@cp "$(COMPLETIONS_DIR)/wiredoor.fish" "usr/share/fish/vendor_completions.d/$(PKG_NAME).fish"
	@cp "$(MAN_GZ_DIR)"/*.gz "usr/share/man/man1/"

build-binaries:
	@mkdir -p $(BIN_PATH)
	@$(foreach arch,$(ARCHS), \
		echo "Building for $(arch)..."; \
		CGO_ENABLED=0 GOOS=linux GOARCH=$(arch) go build \
		-ldflags "-X '$(GO_MODULE)/version.Version=$(VERSION)'" \
		-o $(BIN_PATH)/$(PKG_NAME)-linux-$(arch);)

build-deb:
	@$(foreach arch,$(ARCHS), \
		fpm -s dir -t deb -v $(VERSION) -a $(arch) \
			--depends iptables \
			--depends wireguard-tools \
			--depends iproute2 \
			-p $(OUT_PATH)/$(PKG_NAME)_$(VERSION)-1_debian_$(arch).deb \
			$(BIN_PATH)/$(PKG_NAME)-linux-$(arch)=/usr/bin/$(PKG_NAME) \
			usr=/usr \
			build/linux/etc/system/systemd/$(PKG_NAME).service=/lib/systemd/system/$(PKG_NAME).service;)

build-rpm:
	@$(foreach arch,$(ARCHS), \
		fpm -s dir -t rpm -v $(VERSION) -a $(arch) \
			--depends iptables \
			--depends wireguard-tools \
			--depends iproute \
			-p $(OUT_PATH)/$(PKG_NAME)_$(VERSION)-1_rpm_$(arch).rpm \
			$(BIN_PATH)/$(PKG_NAME)-linux-$(arch)=/usr/bin/$(PKG_NAME) \
			usr=/usr \
			build/linux/etc/system/systemd/$(PKG_NAME).service=/usr/lib/systemd/system/$(PKG_NAME).service;)

build-apk:
	@$(foreach arch,$(ARCHS), \
		fpm -s dir -t apk -v $(VERSION) -a $(arch) \
			--depends iptables \
			--depends wireguard-tools \
			--depends iproute2 \
			-p $(OUT_PATH)/$(PKG_NAME)_$(VERSION)-1_alpine_$(arch).apk \
			$(BIN_PATH)/$(PKG_NAME)-linux-$(arch)=/usr/bin/$(PKG_NAME) \
			usr=/usr \
			build/linux/etc/init.d/$(PKG_NAME).init=/etc/init.d/$(PKG_NAME);)

build-pacman:
	@$(foreach arch,$(ARCHS), \
		fpm -s dir -t pacman -v $(VERSION) -a $(arch) \
			--depends wireguard-tools \
			--depends iproute2 \
			-p $(OUT_PATH)/$(PKG_NAME)_$(VERSION)-1_archlinux_$(arch).pkg.tar.zst \
			$(BIN_PATH)/$(PKG_NAME)-linux-$(arch)=/usr/bin/$(PKG_NAME) \
			usr=/usr \
			build/linux/etc/system/systemd/$(PKG_NAME).service=/usr/lib/systemd/system/$(PKG_NAME).service;)

build-windows:
	@mkdir -p $(BIN_PATH)
	@$(foreach arch,$(WIN_ARCHS), \
		echo "Building Windows for $(arch)..."; \
		CGO_ENABLED=0 GOOS=windows GOARCH=$(arch) go build \
			-ldflags "-X '$(GO_MODULE)/version.Version=$(VERSION)'" \
			-o $(BIN_PATH)/$(PKG_NAME)-windows-$(arch).exe \
			.; \
		cp $(BIN_PATH)/$(PKG_NAME)-windows-$(arch).exe $(OUT_PATH)/$(PKG_NAME)_$(VERSION)_windows_$(arch).exe; )

build-macos:
	@mkdir -p "$(BIN_PATH)" "$(OUT_PATH)"
	@$(foreach arch,$(ARCHS), \
		echo "Building macOS for $(arch)..."; \
		CGO_ENABLED=0 GOOS=darwin GOARCH=$(arch) go build \
			-ldflags "-X '$(GO_MODULE)/version.Version=$(VERSION)'" \
			-o "$(BIN_PATH)/$(PKG_NAME)-darwin-$(arch)" \
			.; \
		echo "Packaging macOS tar.gz for $(arch)..."; \
		rm -rf "$(MACOS_TMP)"; \
		mkdir -p "$(MACOS_TMP)/completions" "$(MACOS_TMP)/man"; \
		cp "$(BIN_PATH)/$(PKG_NAME)-darwin-$(arch)" "$(MACOS_TMP)/$(PKG_NAME)"; \
		chmod +x "$(MACOS_TMP)/$(PKG_NAME)"; \
		cp -R "$(COMPLETIONS_DIR)/." "$(MACOS_TMP)/completions/"; \
		cp -R "$(MAN_GZ_DIR)/." "$(MACOS_TMP)/man/"; \
		( cd "$(MACOS_TMP)" && tar -czf "$(abspath $(OUT_PATH))/$(PKG_NAME)_$(VERSION)_darwin_$(arch).tar.gz" . ); \
		echo "$(PKG_NAME)_$(VERSION)_darwin_$(arch).tar.gz"; \
		shasum -a 256 "$(OUT_PATH)/$(PKG_NAME)_$(VERSION)_darwin_$(arch).tar.gz"; \
	)

clean:
	@rm -rf "$(BIN_PATH)" "$(MACOS_TMP)" "$(COMPLETIONS_DIR)" "$(MAN_DIR)" "$(MAN_GZ_DIR)" "usr"
