PATTERN_TARGETS += test

.PHONY: test
test: hugo
	npx vitest run
