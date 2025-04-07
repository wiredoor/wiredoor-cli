VERSION ?= 1.0.0
OUT_PATH ?= dist

build-pkgs: build-deb build-rpm build-apk

build-deb:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/wiredoor/wiredoor-cli/cmd.Version=${VERSION}'" -o bin/wiredoor-linux-amd64
	fpm -s dir -t deb -v ${VERSION} -a amd64 \
		--depends iptables \
		--depends wireguard-tools \
		--depends iproute2 \
		-p ${OUT_PATH}/wiredoor_${VERSION}-1_debian_amd64.deb \
		bin/wiredoor-linux-amd64=/usr/bin/wiredoor \
		etc/system/systemd/wiredoor.service=/lib/systemd/system/wiredoor.service
	GOOS=linux GOARCH=arm64 go build -ldflags "-X 'github.com/wiredoor/wiredoor-cli/cmd.Version=${VERSION}'" -o bin/wiredoor-linux-arm64
	fpm -s dir -t deb -v ${VERSION} -a arm64 \
		--depends iptables \
		--depends wireguard-tools \
		--depends iproute2 \
		-p ${OUT_PATH}/wiredoor_${VERSION}-1_debian_arm64.deb \
		bin/wiredoor-linux-arm64=/usr/bin/wiredoor \
		etc/system/systemd/wiredoor.service=/lib/systemd/system/wiredoor.service

build-rpm:
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/wiredoor/wiredoor-cli/cmd.Version=${VERSION}'" -o bin/wiredoor-linux-amd64
	fpm -s dir -t rpm -v ${VERSION} -a amd64 \
		--depends iptables \
		--depends wireguard-tools \
		--depends iproute \
		-p ${OUT_PATH}/wiredoor_${VERSION}-1_rpm_amd64.rpm \
		bin/wiredoor-linux-amd64=/usr/bin/wiredoor \
		etc/system/systemd/wiredoor.service=/usr/lib/systemd/system/wiredoor.service
	GOOS=linux GOARCH=arm64 go build -ldflags "-X 'github.com/wiredoor/wiredoor-cli/cmd.Version=${VERSION}'" -o bin/wiredoor-linux-arm64
	fpm -s dir -t deb -v ${VERSION} -a arm64 \
		--depends iptables \
		--depends wireguard-tools \
		--depends iproute \
		-p ${OUT_PATH}/wiredoor_${VERSION}-1_rpm_arm64.rpm \
		bin/wiredoor-linux-arm64=/usr/bin/wiredoor \
		etc/system/systemd/wiredoor.service=/usr/lib/systemd/system/wiredoor.service

build-apk:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/wiredoor/wiredoor-cli/cmd.Version=${VERSION}'" -o bin/wiredoor-linux-amd64
	fpm -s dir -t apk -v ${VERSION} -a amd64 \
		--depends iptables \
		--depends wireguard-tools \
		--depends iproute2 \
		-p ${OUT_PATH}/wiredoor_${VERSION}-1_alpine_amd64.apk \
		bin/wiredoor-linux-amd64=/usr/bin/wiredoor \
		etc/init.d/wiredoor.init=/etc/init.d/wiredoor
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X 'github.com/wiredoor/wiredoor-cli/cmd.Version=${VERSION}'" -o bin/wiredoor-linux-arm64
	fpm -s dir -t apk -v ${VERSION} -a arm64 \
		--depends iptables \
		--depends wireguard-tools \
		--depends iproute2 \
		-p ${OUT_PATH}/wiredoor_${VERSION}-1_alpine_arm64.apk \
		bin/wiredoor-linux-arm64=/usr/bin/wiredoor \
		etc/init.d/wiredoor.init=/etc/init.d/wiredoor
