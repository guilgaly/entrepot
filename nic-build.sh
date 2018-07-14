#! /bin/bash

set -e

DIR=`dirname $0`

if [ "x$DIR" = "x." ]; then
  DIR="$PWD"
fi

# Prepare
if [ ! -d "$DIR/dist" ]; then
  mkdir "$DIR/dist"
fi

# Build
GOPATH=$DIR/src/server/ GOOS=linux GOARCH=386 go build -v -o "$DIR/dist/server" "$DIR"/src/server/*.go

# Copy resources
cp "$DIR/src/Procfile" "$DIR/dist/"
cp -R "$DIR/resources" "$DIR/dist/resources/"

# Archive
cd "$DIR/dist"
zip "$DIR/entrepot.zip" -r .
echo "Distribution ready at $DIR/entrepot.zip"
