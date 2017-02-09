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
	env GOOS=linux GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=linux -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ${BINARY}
	go get -u -v github.com/jteeuwen/go-bindata/...
	cd server && go-bindata static/ login.html post.html
	cd server && env GOOS=linux GOARCH=amd64 \
		go build -o ../${BINARY}server
	cd tool && env GOOS=linux GOARCH=amd64 \
		go build -o ../${BINARY}tool


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
	echo "Bundle data"
	go get -u -v github.com/jteeuwen/go-bindata/...
	cd server && go-bindata static/ login.html post.html
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
	echo "Making Windows-AMD64"
	env GOOS=windows GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=win64 -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ${BINARY}.exe
	cd server && env GOOS=windows GOARCH=amd64 \
		go build -o ../${BINARY}server.exe
	cd tool && env GOOS=windows GOARCH=amd64 \
		go build -o ../${BINARY}tool.exe
	zip -j ${BINARY}-${VERSION}-win64.zip ${BINARY}.exe ${BINARY}tool.exe ${BINARY}server.exe README.md LICENSE
	github-release upload \
				--user ${USER} \
				--repo ${BINARY} \
				--tag ${VERSION} \
				--name "${BINARY}-${VERSION}-win64.zip" \
				--file ${BINARY}-${VERSION}-win64.zip
	echo "Making Linux-AMD64"
	env GOOS=linux GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=linux -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ${BINARY}
	cd server && env GOOS=linux GOARCH=amd64 \
		go build -o ../${BINARY}server
	cd tool && env GOOS=linux GOARCH=amd64 \
		go build -o ../${BINARY}tool
	zip -j ${BINARY}-${VERSION}-linux64.zip ${BINARY} ${BINARY}tool ${BINARY}server README.md LICENSE
	github-release upload \
				--user ${USER} \
				--repo ${BINARY} \
				--tag ${VERSION} \
				--name "${BINARY}-${VERSION}-linux64.zip" \
				--file ${BINARY}-${VERSION}-linux64.zip
	echo "Making OSX-AMD64"
	env GOOS=darwin GOARCH=amd64 \
		go build -ldflags \
		"-X main.OS=osx -X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}" \
		-o ${BINARY}
	cd server && env GOOS=darwin GOARCH=amd64 \
		go build -o ../${BINARY}server
	cd tool && env GOOS=darwin GOARCH=amd64 \
		go build -o ../${BINARY}tool
	zip -j ${BINARY}-${VERSION}-osx.zip ${BINARY} ${BINARY}tool ${BINARY}server README.md LICENSE
	github-release upload \
				--user ${USER} \
				--repo ${BINARY} \
				--tag ${VERSION} \
				--name "${BINARY}-${VERSION}-osx.zip" \
				--file ${BINARY}-${VERSION}-osx.zip
