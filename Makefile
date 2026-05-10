# Based on that awesome makefile https://github.com/dunglas/symfony-docker/blob/main/docs/makefile.md#the-template

# Sed in-place option behaves differently on linux and macOs
ifeq ($(shell uname),Darwin)
    SED_INPLACE_OPTION=-i ''
else
    SED_INPLACE_OPTION=-i
endif

.DEFAULT_GOAL = default

.PHONY: default
default: build

##—— 📚 Help ———————————————————————————————————————————————————————————
.PHONY: help
help: ## ❓ Display this help
	@grep -E '(^[a-zA-Z0-9_-]+:.*?##.*$$)|(^##)' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}{printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' \
		| sed -e 's/\[32m##——————————/[33m           /'  \
		| sed -e 's/\[32m##——/[33m ——/' \
		| sed -e 's/\[32m####/[34m                                 /' \
		| sed -e 's/\[32m###/[36m                                 /' \
		| sed -e 's/\[32m##\?/[35m /'  \
		| sed -e 's/\[32m##/[33m/'

##—— ️⚙️  Environments ——————————————————————————————————————————————————————
.PHONY: configure-dev-env
configure-dev-env: ## 🤖 Install required libraries for dev environment
configure-dev-env:
	go install github.com/posener/goreadme/cmd/goreadme@v1

.PHONY: configure-test-env
configure-test-env: ## 🤖 Install required libraries for test environment (golint, staticcheck, etc)
configure-test-env: configure-dev-env
configure-test-env:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11

##—— 📝 Documentation —————————————————————————————————————————————
.PHONY: build-doc
.SILENT: build-doc
build-doc: ## 🗜️  Build packages doc
build-doc:
	echo "Generate doc for main package ..."
	goreadme -constants -variabless -types -methods -functions -factories -recursive > DOC.md

##—— 🐹 Golang —————————————————————————————————————————————
.PHONY: build
build: ## 🗜️  Build package
#### Use build_o="..." to specify build options
$(eval build_o ?=)
build:
	go build -v $(build_o) ./cmd/bench-reg...

.PHONY: verify-deps
verify-deps: ## 🗜️  Verify dependencies
verify-deps:
	go mod verify

##—— 🧪️ Tests —————————————————————————————————————————————————————————
.PHONY: test
test: ## 🏃 Launch all tests
test: test-go test-lint

.PHONY: test-go
test-go: ## 🏃 Launch go test
#### Use gotest_o="..." to specify options
$(eval gotest_o ?=)
test-go:
	go test -v $(gotest_o) ./...

.PHONY: test-lint
test-lint: ## 🏃 Launch golangci-lint
#### Use lint_o="..." to specify options
$(eval lint_o ?=--fix)
test-lint:
	golangci-lint run $(lint_o) ./...

##—— 📋 Code Quality —————————————————————————————————————————————
.PHONY: fmt
fmt: ## 🔧 Format code with go fmt
	go fmt ./...

.PHONY: vet
vet: ## 🔍 Run go vet (suspicious code patterns)
	go vet ./...

.PHONY: benchmark
#### Use bench_o="..." to specify options
$(eval bench_o ?=)
benchmark: ## 🔍 Run benchmarks
	go test -run='^$$' -bench=. -benchmem $(bench_o) ./...
