goimports := golang.org/x/tools/cmd/goimports@v0.7.0
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.1

.PHONY: build
build:
	@find ./filters -type f -name "main.go" | grep ${target}\
	| xargs -I {} bash -c 'dirname {}' \
	| xargs -I {} bash -c 'cd {} && tinygo build -o main.wasm -scheduler=none -target=wasi ./main.go'

.PHONY: lint
lint:
	@find . -name "go.mod" \
	| grep go.mod \
	| xargs -I {} bash -c 'dirname {}' \
	| xargs -I {} bash -c 'echo "=> {}"; cd {}; go run $(golangci_lint) run; '

.PHONY: format
format:
	@find . -type f -name '*.go' | xargs gofmt -s -w
	@for f in `find . -name '*.go'`; do \
	    awk '/^import \($$/,/^\)$$/{if($$0=="")next}{print}' $$f > /tmp/fmt; \
	    mv /tmp/fmt $$f; \
	done
	@go run $(goimports) -w -local github.com/tetratelabs/proxy-wasm-go-sdk `find . -name '*.go'`

.PHONY: check
check:
	@$(MAKE) format
	@go mod tidy
	@if [ ! -z "`git status -s`" ]; then \
		echo "The following differences will fail CI until committed:"; \
		git diff --exit-code; \
	fi

.PHONY: tidy
tidy: ## Runs go mod tidy on every module
	@find . -name "go.mod" \
	| grep go.mod \
	| xargs -I {} bash -c 'dirname {}' \
	| xargs -I {} bash -c 'echo "=> {}"; cd {}; go mod tidy -v; '
