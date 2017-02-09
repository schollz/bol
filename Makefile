SOURCEDIR=.

BINARY=bol
USER=schollz
VERSION=0.1.2
BUILD_TIME=`date +%FT%T%z`
BUILD=`git rev-parse HEAD`
BUILDSHORT = `git rev-parse --short HEAD`
OS=something
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}"

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go build -ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME} -X main.OS=linux_amd64" -o ${BINARY}

.PHONY: build
build:
	# echo "Bundle data"
	# go get -u -v github.com/jteeuwen/go-bindata/...
	# cd server && go-bindata static/ login.html post.html
	mkdir -p build
	echo "Building Linux AMD64"
	cd bol && env GOOS=linux GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=linux -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ../build/${BINARY}
	cd bolserver && env GOOS=linux GOARCH=amd64 \
		go build -o ../build/${BINARY}server
	cd boltool && env GOOS=linux GOARCH=amd64 \
		go build -o ../build/${BINARY}tool
	cd build && zip -j ${BINARY}-${VERSION}-linux64.zip ${BINARY} ${BINARY}tool ${BINARY}server README.md LICENSE
	rm build/boltool build/bol build/bolserver
	echo "Building windows AMD64"
	cd bol && env GOOS=windows GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=win64 -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ../build/${BINARY}.exe
	cd bolserver && env GOOS=windows GOARCH=amd64 \
		go build -o ../build/${BINARY}server.exe
	cd boltool && env GOOS=windows GOARCH=amd64 \
		go build -o ../build/${BINARY}tool.exe
	cd build && zip -j ${BINARY}-${VERSION}-win64.zip ${BINARY}.exe ${BINARY}tool.exe ${BINARY}server.exe README.md LICENSE
	rm build/boltool.exe build/bol.exe build/bolserver.exe

.PHONY: delete
delete:
	echo "Deleting release ${VERSION}"
	git tag -d ${VERSION};
	git push origin :${VERSION};
	github-release delete \
			--user ${USER} \
			--repo ${BINARY} \
			--tag ${VERSION}

.PHONY: release
release:
	mkdir -p build
	echo "Moving tag"
	git tag --force latest ${BUILD}
	git push --force --tags
	echo "Creating new release ${VERSION}"
	github-release release \
	    --user ${USER} \
	    --repo ${BINARY} \
	    --tag ${VERSION} \
	    --name "${VERSION}" \
	    --description "This is a standalone latest of ${BINARY}."
	echo "Making Linux-AMD64"
	github-release upload \
				--user ${USER} \
				--repo ${BINARY} \
				--tag ${VERSION} \
				--name "${BINARY}-${VERSION}-linux64.zip" \
				--file build/${BINARY}-${VERSION}-linux64.zip
	echo "Making Windows-AMD64"
	github-release upload \
				--user ${USER} \
				--repo ${BINARY} \
				--tag ${VERSION} \
				--name "${BINARY}-${VERSION}-win64.zip" \
				--file build/${BINARY}-${VERSION}-win64.zip
