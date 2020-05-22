ARCH?=amd64
OS?=linux

PKG=$(shell go list -m)

.EXPORT_ALL_VARIABLES:

build:
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags "-X $(PKG)/version.COMMIT=${GIT_COMMIT} -X $(PKG)/version.RELEASE=${GIT_TAG} -X $(PKG)/version.REPO=${GIT_REPO}" -o cloudflare-access-controller ./cmd

docker:
	docker build -t dingggu/cloudflare-access-controller .