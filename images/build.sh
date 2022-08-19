#!/usr/bin/env sh

cp -R /handlers .

mv main.go.templ main.go

go mod tidy

CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -installsuffix cgo -o server .

cp server /server/