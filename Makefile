VERSION ?= 1.0.0
OUT_PATH ?= dist
BIN_PATH := bin
PKG_NAME := wiredoor
GO_MODULE := github.com/wiredoor/wiredoor-cli
ARCHS := amd64 arm64

build-pkgs: build-artifacts build-binaries build-deb build-rpm build-apk build-pacman build-windows

build-artifacts:
	chmod +x ./gen-winres.sh
	./gen-winres.sh

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
			etc/system/systemd/$(PKG_NAME).service=/lib/systemd/system/$(PKG_NAME).service;)

build-rpm:
	@$(foreach arch,$(ARCHS), \
		fpm -s dir -t rpm -v $(VERSION) -a $(arch) \
			--depends iptables \
			--depends wireguard-tools \
			--depends iproute \
			-p $(OUT_PATH)/$(PKG_NAME)_$(VERSION)-1_rpm_$(arch).rpm \
			$(BIN_PATH)/$(PKG_NAME)-linux-$(arch)=/usr/bin/$(PKG_NAME) \
			etc/system/systemd/$(PKG_NAME).service=/usr/lib/systemd/system/$(PKG_NAME).service;)

build-apk:
	@$(foreach arch,$(ARCHS), \
		fpm -s dir -t apk -v $(VERSION) -a $(arch) \
			--depends iptables \
			--depends wireguard-tools \
			--depends iproute2 \
			-p $(OUT_PATH)/$(PKG_NAME)_$(VERSION)-1_alpine_$(arch).apk \
			$(BIN_PATH)/$(PKG_NAME)-linux-$(arch)=/usr/bin/$(PKG_NAME) \
			etc/init.d/$(PKG_NAME).init=/etc/init.d/$(PKG_NAME);)

build-pacman:
	@$(foreach arch,$(ARCHS), \
		fpm -s dir -t pacman -v $(VERSION) -a $(arch) \
			--depends iptables \
			--depends wireguard-tools \
			--depends iproute2 \
			-p $(OUT_PATH)/$(PKG_NAME)_$(VERSION)-1_archlinux_$(arch).pkg.tar.zst \
			$(BIN_PATH)/$(PKG_NAME)-linux-$(arch)=/usr/bin/$(PKG_NAME) \
			etc/system/systemd/$(PKG_NAME).service=/usr/lib/systemd/system/$(PKG_NAME).service;)

build-windows:
	@mkdir -p $(BIN_PATH)
	@$(foreach arch,$(ARCHS), \
		echo "Building Windows for $(arch)..."; \
		CGO_ENABLED=0 GOOS=windows GOARCH=$(arch) go build \
			-ldflags "-X '$(GO_MODULE)/version.Version=$(VERSION)'" \
			-o $(BIN_PATH)/$(PKG_NAME)-windows-$(arch).exe \
			.; \
		cp $(BIN_PATH)/$(PKG_NAME)-windows-$(arch).exe $(OUT_PATH)/$(PKG_NAME)_$(VERSION)_windows_$(arch).exe; )
