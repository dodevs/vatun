#!/bin/bash

BINARY_NAME="vatun"
BINARY_FOLDER="bin"

go build -o $BINARY_FOLDER/$BINARY_NAME cmd/*.go