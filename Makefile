include .env
export $(shell sed 's/=.*//' .env)

.PHONY: up down dev-start dev mcp-servers test-coverage test build-frontend

build-frontend:
	cd frontend && npm install && npm run build

up:
	podman compose up --build

down:
	podman compose down

dev-start:
	podman run --name support_copilot_postgres \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-e POSTGRES_USER=$(DB_USER) \
		-e POSTGRES_DB=$(DB_NAME) \
		-p $(DB_PORT):5432 \
		-d docker.io/library/postgres:15
migrate:
	go run backend/main.go migrate
dev:
	go run -race backend/main.go server

mcp-servers:
	cd mcp_servers && ( [ -d .venv ] && .venv/bin/python main.py || python3 main.py )

MOCKERY = $(shell pwd)/backend/bin/mockery

$(MOCKERY):
	GOBIN=$(shell pwd)/backend/bin go install github.com/vektra/mockery/v2@v2.53.6

generate: $(MOCKERY)
	$(MOCKERY) --all --dir backend/internal/interfaces --output backend/internal/mocks --outpkg mocks --case camel


test-coverage:
	@tmp=$$(mktemp); \
	total_failed=0; \
	cd backend && { \
		for pkg in $$(go list ./... | grep -vE '(mocks|interfaces|types|prompt)'); do \
			output=$$(go test -cover $$pkg 2>&1); \
			status=$$?; \
			coverage=$$(echo "$$output" | grep -oE 'coverage: [0-9.]+%' | awk '{print $$2}' | tr -d '%'); \
			coverage=$${coverage:-0}; \
			echo "$$coverage $$pkg" >> $$tmp; \
			if [ $$status -ne 0 ]; then total_failed=1; fi; \
		done; \
		col_width=$$(awk 'BEGIN { max = 0 } { if (length($$2) > max) max = length($$2) } END { print max }' $$tmp); \
		left_border=$$(printf '%*s' $$((col_width + 2)) '' | tr ' ' '-'); \
		printf "+%s+----------+\n" "$$left_border"; \
		printf "| %-*s | %8s |\n" "$$col_width" "Directory" "Coverage"; \
		printf "+%s+----------+\n" "$$left_border"; \
		sort -n $$tmp | while read cov pkg; do \
			printf "| %-*s | %7.1f%% |\n" "$$col_width" "$$pkg" "$$cov"; \
		done; \
		printf "+%s+----------+\n" "$$left_border"; \
		go test -coverprofile=coverage.out ./... >/dev/null 2>&1; \
		grep -vE '(mocks|interfaces|types|prompt)' coverage.out > coverage_filtered.out 2>/dev/null; \
		total=$$(go tool cover -func=coverage_filtered.out | grep total | awk '{print $$3}'); \
		printf "| %-*s | %8s |\n" "$$col_width" "TOTAL" "$$total"; \
		printf "+%s+----------+\n" "$$left_border"; \
		rm -f $$tmp coverage.out coverage_filtered.out; \
		exit 0; \
	}

test:
	@cd backend && go test -v -cover ./...
