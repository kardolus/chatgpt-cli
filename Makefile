# Default goal when running `make`
.DEFAULT_GOAL := help

.PHONY: help all-tests binaries commit contract install integration mdrender reinstall run_test shipit unit updatedeps

# Help command to list all available targets
help:  ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

all-tests: ## Run all tests, including linting, formatting, and 'go mod tidy'
	./scripts/all-tests.sh

binaries: ## Create binaries for multiple platforms
	./scripts/binaries.sh

commit: ## Generate a commit message using ChatGPT based on git diff
	git status -vv | chatgpt -n -p ../prompts/create_git_diff_commit.md

contract: ## Run contract tests
	./scripts/contract.sh

install: ## Build the binaries for the specified OS (default: darwin)
	./scripts/install.sh $(TARGET_OS)

integration: ## Run integration tests
	./scripts/integration.sh

reinstall: ## Reinstall binaries (default target OS: darwin)
	./scripts/reinstall.sh $(TARGET_OS)

run_test: ## Run specified test type (Unit, Integration, Contract)
	./scripts/run_test.sh $(TEST_TYPE)

shipit: ## Run the release script, create binaries, and generate release notes
	./scripts/shipit.sh

unit: ## Run unit tests
	./scripts/unit.sh

updatedeps: ## Update dependencies and vendor them
	./scripts/updatedeps.sh