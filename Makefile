GO ?= go
.PHONY: bootstrap dev-api dev-ui check

bootstrap:
	python3 scripts/import-source.py

dev-api: bootstrap
	cd backend/platform && $(GO) run ./cmd/cognara-dev --demo

dev-ui: bootstrap
	cd frontend && npm ci && npm run dev

check: bootstrap
	cd backend/platform && GOGC=30 GOMAXPROCS=2 $(GO) test -tags=nomsgpack -p 1 -race ./internal/platform/... ./cmd/cognara-dev
	cd frontend && npm run test:cognara && npx vue-tsc --noEmit -p tsconfig.cognara.json && npm run build
