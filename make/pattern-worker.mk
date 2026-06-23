PATTERN_TARGETS += test

.PHONY: test
test:
	npx vitest run
