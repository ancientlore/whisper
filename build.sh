#!/bin/bash

GO_VERSION=$(go env GOVERSION | cut -b "3-")
GO_MAJOR_VERSION=$(cut -d '.' -f 1,2 <<< "$GO_VERSION")
TAG=$(git tag | sort -V | tail -1)

echo
echo Go version is $GO_VERSION, major version is $GO_MAJOR_VERSION
echo Tag is $TAG

echo
echo Building ancientlore/whisper:$TAG
docker buildx build --build-arg GO_VERSION=$GO_VERSION --build-arg IMG_VERSION=$GO_MAJOR_VERSION --platform linux/amd64,linux/arm64 -t ancientlore/whisper:$TAG . || exit 1

gum confirm "Push?" || exit 1

echo
echo Pushing ancientlore/whisper:$TAG
docker push ancientlore/whisper:$TAG || exit 1

echo
echo Tagging ancientlore/whisper:latest
docker tag ancientlore/whisper:$TAG ancientlore/whisper:latest || exit 1

echo
echo Pushing ancientlore/whisper:latest
docker push ancientlore/whisper:latest || exit 1

echo
echo Tagging ancientlore.registry.cpln.io/whisper:$TAG
docker tag ancientlore/whisper:$TAG ancientlore.registry.cpln.io/whisper:$TAG || exit 1

echo
echo Pushing ancientlore.registry.cpln.io/whisper:$TAG
docker push ancientlore.registry.cpln.io/whisper:$TAG || exit 1
