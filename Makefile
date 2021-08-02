VERSION=`git rev-parse --short HEAD`
flags=-ldflags="-s -w -X web.version=${VERSION}"
odir=`cat ${PKG_CONFIG_PATH}/oci8.pc | grep "libdir=" | sed -e "s,libdir=,,"`

all: build

vet:
	go vet .

build:
	go clean; rm -rf pkg dbsproxy*; go build ${flags}

build_all: build build_darwin build_amd64 build_power8 build_arm64

build_darwin:
	go clean; rm -rf pkg dbsproxy_darwin; GOOS=darwin go build ${flags}
	mv dbsproxy dbsproxy_darwin

build_amd64:
	go clean; rm -rf pkg dbsproxy_amd64; GOOS=linux go build ${flags}
	mv dbsproxy dbsproxy_amd64

build_power8:
	go clean; rm -rf pkg dbsproxy_power8; GOARCH=ppc64le GOOS=linux go build ${flags}
	mv dbsproxy dbsproxy_power8

build_arm64:
	go clean; rm -rf pkg dbsproxy_arm64; GOARCH=arm64 GOOS=linux go build ${flags}
	mv dbsproxy dbsproxy_arm64

install:
	go install

clean:
	go clean; rm -rf pkg
