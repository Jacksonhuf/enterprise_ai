GO ?= go
.PHONY: bootstrap dev-api dev-ui check

bootstrap:
	python3 scripts/import-source.py

dev-api: bootstrap
	cd backend/platform && $(GO) run ./cmd/cognara-dev --demo

dev-ui: bootstrap
	cd frontend && npm ci && npm run dev

check: bootstrap
	cd backend/platform && GOGC=30 GOMAXPROCS=2 $(GO) test -tags=nomsgpack -p 1 -race ./internal/platform/... ./cmd/cognara-dev ./cmd/cognara-migrate-applications
	cd frontend && npm run test:cognara && npx vue-tsc --noEmit -p tsconfig.cognara.json && npm run build

.PHONY: check-postgres build-dev verify-runtime verify-postgres
check-postgres: bootstrap
	@test -n "$${COGNARA_TEST_POSTGRES_DSN}" || (echo 'Set a dedicated COGNARA_TEST_POSTGRES_DSN' >&2; exit 1)
	cd backend/platform && COGNARA_REQUIRE_POSTGRES_TESTS=1 GOGC=30 GOMAXPROCS=2 $(GO) test -tags=nomsgpack -p 1 -race -count=1 ./internal/platform/... ./cmd/cognara-dev ./cmd/cognara-migrate-applications

build-dev: bootstrap
	mkdir -p backend/platform/.devdata
	cd backend/platform && $(GO) build -tags=nomsgpack -o .devdata/cognara-dev ./cmd/cognara-dev

verify-runtime: build-dev
	python3 scripts/verify-runtime.py --binary backend/platform/.devdata/cognara-dev --storage json
	python3 scripts/verify-runtime.py --binary backend/platform/.devdata/cognara-dev --storage sqlite

verify-postgres: build-dev
	@test -n "$${COGNARA_TEST_POSTGRES_DSN}" || (echo 'Set a dedicated COGNARA_TEST_POSTGRES_DSN' >&2; exit 1)
	COGNARA_POSTGRES_DSN="$${COGNARA_TEST_POSTGRES_DSN}" python3 scripts/verify-runtime.py --binary backend/platform/.devdata/cognara-dev --storage postgres
