default:
    @just --list

# Build the fzfg binary to ./dist/fzfg
build:
    go build -ldflags='-s -w' -o ./dist/fzfg ./cmd/fzfg/

# Run fzfg with arguments (e.g., just run -c fd_files -o preview)
run *ARGS:
    go run ./cmd/fzfg/ {{ ARGS }}

# Run all tests (Go unit tests + functional shell tests)
test:
    go test --json -v ./... 2>&1 |tee /tmp/gotest.log | gotestfmt 
    ./scripts/tests

# Run Go unit tests only
test-unit:
    gotestsum --format testname

# go test fmt -json | tparse -all

# Build and install to ~/.local/bin, copy configs to ~/.config/fzfg
install: build
    [ -d "$HOME/.local/bin" ] && mv -v ./dist/fzfg "$HOME/.local/bin/fzfg"
    mkdir -pv "$HOME/.config/fzfg"
    cp -av ./configs/* "$HOME/.config/fzfg"

# Run golangci-lint
lint:
    golangci-lint run ./...

lint-verbose:
    golangci-lint run --verbose ./...

# Format Go source files
fmt:
    gofmt -w .
