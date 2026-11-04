.PHONY: run test vet build release runtime-test

run:
	go run -tags 'production webkit2_41' ./cmd/podcasts3000

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -tags 'production webkit2_41' -o podcasts3000 ./cmd/podcasts3000

release:
	mkdir -p release
	GOOS=linux GOARCH=amd64 go build -tags 'production webkit2_41' -o release/podcasts3000-linux-amd64 ./cmd/podcasts3000
	GOOS=windows GOARCH=amd64 go build -tags production -o release/podcasts3000-windows-amd64.exe ./cmd/podcasts3000

run-dev:
	go run -tags 'dev webkit2_41' ./cmd/podcasts3000

runtime-test:
	go test -tags 'production webkit2_41' ./cmd/podcasts3000 ./internal/podcasts
