#!/bin/sh

make -C Aurora
go build .

./build-flatpak.sh