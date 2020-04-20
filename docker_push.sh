#!/bin/bash

REVISION=$(git describe --tags)
VERSION=$(echo $REVISION | cut -d'.' -f 1,2,3)
REPOSITORY="luciaptech/chockagent"
TAG="$VERSION-alpine3.11"

echo "Building $REPOSITORY:$TAG"
docker build -t $REPOSITORY .
docker tag "$REPOSITORY" "$REPOSITORY:$TAG"

echo "Publishing $REPOSITORY:$TAG"
docker login
docker push "$REPOSITORY:$TAG"
docker push "$REPOSITORY:latest"
