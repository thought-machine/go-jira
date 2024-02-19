.DEFAULT_GOAL := help

.PHONY: help
help: ## Outputs the help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Runs all unit, integration and example tests.
	go test -race -v ./...

.PHONY: all
all: test
