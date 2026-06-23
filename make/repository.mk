TOP_LEVEL := $(shell git rev-parse --show-toplevel)
PATTERN := $(shell cat $(TOP_LEVEL)/.pattern)
