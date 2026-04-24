.PHONY: help dev smoke up down logs lint test proto contracts clean

help:
	@echo "EvidenceLens — common targets"
	@echo "  make dev        Bring up local docker-compose stack (data services + apps)"
	@echo "  make smoke      End-to-end smoke test against running stack"
	@echo "  make up         Start docker-compose detached"
	@echo "  make down       Stop docker-compose"
	@echo "  make logs       Tail docker-compose logs"
	@echo "  make proto      Run buf lint + buf generate"
	@echo "  make contracts  Build @evidencelens/contracts package"
	@echo "  make lint       Lint all languages (go, python, ts, proto)"
	@echo "  make test       Run unit tests in all workspaces"
	@echo "  make clean      Remove build artifacts and caches"

dev: up

up:
	docker compose -f infra/docker-compose.yml up -d

down:
	docker compose -f infra/docker-compose.yml down

logs:
	docker compose -f infra/docker-compose.yml logs -f --tail=100

smoke:
	@echo "TODO: implement end-to-end smoke (Stream E exit criterion)"
	@exit 1

proto:
	cd proto && buf lint && buf generate

contracts: proto
	pnpm -F @evidencelens/contracts build

lint:
	@echo "==> proto"
	cd proto && buf lint
	@echo "==> go"
	go work sync && cd ingest && golangci-lint run ./... && cd ../index && golangci-lint run ./...
	@echo "==> python"
	cd process && uv run ruff check . && uv run mypy .
	cd embedder && uv run ruff check . && uv run mypy .
	cd scorer && uv run ruff check . && uv run mypy .
	cd agent && uv run ruff check . && uv run mypy .
	@echo "==> ts"
	pnpm -r lint

test:
	@echo "==> go"
	cd ingest && go test ./... && cd ../index && go test ./...
	@echo "==> python"
	cd process && uv run pytest && cd ../embedder && uv run pytest
	cd ../scorer && uv run pytest && cd ../agent && uv run pytest
	@echo "==> ts"
	pnpm -r test

clean:
	rm -rf node_modules */node_modules */dist */build */__pycache__ */.next */.venv
	cd ingest && go clean -cache -testcache ./...
	cd index && go clean -cache -testcache ./...
