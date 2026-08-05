.PHONY: fmt vet test tidy check

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./... -v

tidy:
	go mod tidy

check: fmt vet test