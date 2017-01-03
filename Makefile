SOURCEDIR=.

BINARY=bol
USER=schollz
VERSION=0.1.0
BUILD_TIME=`date +%FT%T%z`
BUILD=`git rev-parse HEAD`
BUILDSHORT = `git rev-parse --short HEAD`
OS=something
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}"

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go build -ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME} -X main.OS=linux_amd64" -o ${BINARY}


.PHONY: delete
	echo "Deleting release ${VERSION}"
	git tag -d ${VERSION};
	git push origin :${VERSION};
	github-release delete \
			--user ${USER} \
			--repo ${BINARY} \
			--tag ${VERSION}

.PHONY: release
release:
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
	zip -j ${BINARY}-${VERSION}-win64.zip ${BINARY}.exe README.md LICENSE
	github-release upload \
				--user ${USER} \
				--repo ${BINARY} \
				--tag ${VERSION} \
				--name "${BINARY}-${VERSION}-win64.zip" \
				--file ${BINARY}-${VERSION}-win64.zip
