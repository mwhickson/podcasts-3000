.PHONY: run test vet build runtime-test

run:
	go run -tags 'production webkit2_41' ./cmd/podcasts3000

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -tags 'production webkit2_41' -o podcasts3000 ./cmd/podcasts3000

run-dev:
	go run -tags 'dev webkit2_41' ./cmd/podcasts3000

runtime-test:
	go test -tags 'production webkit2_41' ./cmd/podcasts3000 ./internal/podcasts
