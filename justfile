#!/usr/bin/env -S just --justfile

[group: 'build']
test:
	go test -v ./...

[group: 'build']
build: test
	CGO_ENABLED=0 go build -o bin/corsway cmd/corsway/main.go

[group: 'build']
build-docker: test
	docker build -t axiom-nz/corsway .

[group: 'run']
run port='8080' *args: build
    ./bin/corsway -port {{port}} {{args}}

[group: 'run']
run-docker port='8080' *args: build-docker
    docker run --rm -p {{port}}:{{port}} axiom-nz/corsway -port {{port}} {{args}}
