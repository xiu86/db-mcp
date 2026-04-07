.PHONY: build test test-unit test-integration lint clean

build:
	go build -o bin/db-mcp ./cmd/server

test:
	go test ./... -v -coverprofile=coverage.out

test-unit:
	go test ./tests/unit/... -v -cover

test-integration:
	go test ./tests/integration/... -v -cover

test-coverage:
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total

coverage-check:
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$$(echo "$$coverage >= 100" | bc)" -eq 1 ]; then \
		echo "Coverage: $$coverage%"; \
	else \
		echo "Coverage: $$coverage% (required: 100%)"; \
		exit 1; \
	fi

lint:
	golangci-lint run

clean:
	rm -rf bin/ coverage.out coverage.html
