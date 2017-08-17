BINARY=trafficrouter
UNAME_S=$(shell uname -s | tr [:upper:] [:lower:])
UNAME_M=$(shell uname -m)
GOOS ?= $(UNAME_S)
VERSION ?= latest


ifeq ($(UNAME_M),i386)
  GOARCH ?= 386
endif
ifeq ($(UNAME_M),x86_64)
  GOARCH ?= amd64
endif

ifeq ($(GOARCH),386)
  ARCH=i386
endif
ifeq ($(GOARCH),amd64)
  ARCH=x86_64
endif

ifeq ($(GOOS),linux)
  OS=Linux
endif
ifeq ($(GOOS),darwin)
  OS=Darwin
endif

all: trafficrouter

trafficrouter: main.go 
	go build -o release/${OS}/${ARCH}/bin/${BINARY} main.go  
	chmod u+s release/${OS}/${ARCH}/bin/${BINARY}

go-deps:
	go get .

install:
	cp release/${OS}/${ARCH}/bin/${BINARY} /usr/local/bin/${BINARY}

uninstall:
	rm /usr/local/bin/${BINARY}

clean:
	rm -rf release

package: trafficrouter
	cd release/${OS}/${ARCH} && tar -czvf ${BINARY}-${VERSION}.tgz -T ../../../release_files.txt

upload: package
	aws --region us-west-1 s3 cp release/${OS}/${ARCH}/${BINARY}-${VERSION}.tgz s3://get.dupper.co/${BINARY}/builds/${OS}/${ARCH}/${BINARY}-${VERSION}.tgz --acl public-read

release: package
	aws --region us-west-1 s3 cp release/${OS}/${ARCH}/${BINARY}-${VERSION}.tgz s3://get.dupper.co/${BINARY}/release/${OS}/${ARCH}/${BINARY}-${VERSION}.tgz --acl public-read

publish:
	sed -i.bak "s/VERSION=.*/VERSION=${VERSION}/g" install.sh
	aws --region us-west-1 s3 cp --acl public-read --content-type 'text/plain' ./install.sh s3://get.dupper.co/${BINARY}/index 
