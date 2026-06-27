.PHONY: test test-go test-go-junit test-go-race test-desktop

test: test-go test-desktop

test-go:
	go test ./...

test-go-junit:
	bash ./scripts/test-go-junit.sh

test-go-race:
	go test -race ./...

test-desktop:
	npm run test --prefix desktop
