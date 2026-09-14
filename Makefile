.PHONY: build build-linux test validate validate-result schema-check contract lint

build:
	go build -o bin/holdout ./cmd/holdout

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/holdout-linux-amd64 ./cmd/holdout

test:
	go test ./...
	python -m compileall -q tasks
	python tools/oracle_contract.py
	python tools/task_docs_contract.py

contract:
	python tools/contract.py

schema-check:
	python tools/validate_result_schema.py results.json

validate:
	go run ./cmd/holdout validate --suite public-v0

validate-result:
	go run ./cmd/holdout validate-result --file results.json --strict
	python tools/validate_result_schema.py results.json

lint:
	gofmt -d cmd internal
	python -m compileall -q tasks tools
	python -m ruff check tasks tools
	@if command -v shellcheck >/dev/null 2>&1; then shellcheck runner/firecracker/*.sh; else echo "shellcheck not installed; install it for shell lint"; fi
