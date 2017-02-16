SOURCEDIR=.

BINARY=bol
USER=schollz
VERSION=1.0.0
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
	cd bolserver && go-bindata static/* login.html post.html
	rm -rf build
	mkdir -p build
	echo "Building Linux AMD64"
	cd bol && env GOOS=linux GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=linux -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ../build/${BINARY}
	cd bolserver && env GOOS=linux GOARCH=amd64 \
		go build -o ../build/${BINARY}server
	cd build && zip -j ${BINARY}-${VERSION}-linux64.zip ${BINARY} ${BINARY}server ../README.md ../LICENSE
	rm build/bol build/bolserver
	echo "Building windows AMD64"
	cd bol && env GOOS=windows GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=win64 -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ../build/${BINARY}.exe
	cd bolserver && env GOOS=windows GOARCH=amd64 \
		go build -o ../build/${BINARY}server.exe
	cd build && zip -j ${BINARY}-${VERSION}-win64.zip ${BINARY}.exe ${BINARY}server.exe ../README.md ../LICENSE
	rm build/bol.exe build/bolserver.exe
	echo "Building OSX AMD64"
	cd bol && env GOOS=darwin GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=win64 -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ../build/${BINARY}
	cd bolserver && env GOOS=darwin GOARCH=amd64 \
		go build -o ../build/${BINARY}server
	cd build && zip -j ${BINARY}-${VERSION}-osx.zip ${BINARY} ${BINARY}server ../README.md ../LICENSE
	rm build/bol build/bolserver
	echo "Building ARM"
	cd bol && env GOOS=linux GOARCH=arm \
		go build -ldflags \
		"-X main.OS=win64 -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ../build/${BINARY}
	cd bolserver && env GOOS=linux GOARCH=arm \
		go build -o ../build/${BINARY}server
	cd build && zip -j ${BINARY}-${VERSION}-arm.zip ${BINARY} ${BINARY}server ../README.md ../LICENSE
	rm build/bol build/bolserver

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
	echo "Making OSX"
	github-release upload \
				--user ${USER} \
				--repo ${BINARY} \
				--tag ${VERSION} \
				--name "${BINARY}-${VERSION}-osx.zip" \
				--file build/${BINARY}-${VERSION}-osx.zip
	echo "Making Linux AFM"
	github-release upload \
				--user ${USER} \
				--repo ${BINARY} \
				--tag ${VERSION} \
				--name "${BINARY}-${VERSION}-arm.zip" \
				--file build/${BINARY}-${VERSION}-arm.zip

	git pull
