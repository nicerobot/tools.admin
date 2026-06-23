PATTERN_TARGETS ?=

.PHONY: help
help:
	@echo "Available targets: help hugo dev build deploy status init$(if $(PATTERN_TARGETS), $(PATTERN_TARGETS))"

.PHONY: hugo
hugo:
	hugo --minify

.PHONY: dev
dev: hugo
	op-run npx wrangler dev

.PHONY: build
build:
	op-run npx wrangler deploy --dry-run

.PHONY: deploy
deploy:
	op-run npx wrangler deploy

.PHONY: status
status:
	op-run npx wrangler deployments list
