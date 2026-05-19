.DEFAULT_GOAL := help

CARGO ?= cargo
ARGS ?=

.PHONY: help tools fmt fmt-check check clippy test test-doc build production release prod dev verify ci run clean

help:
	@printf "lnk development targets\n"
	@printf "\n"
	@printf "  make dev            Format, type-check, lint, and run nextest\n"
	@printf "  make verify         Run the full required gate before finalizing changes\n"
	@printf "  make tools          Print required tool versions\n"
	@printf "\n"
	@printf "  make fmt            Format Rust sources\n"
	@printf "  make fmt-check      Check Rust formatting\n"
	@printf "  make check          Type-check all targets and features\n"
	@printf "  make clippy         Lint all targets and features with warnings denied\n"
	@printf "  make test           Run unit and integration tests with nextest\n"
	@printf "  make test-doc       Run doctests\n"
	@printf "  make build          Build all targets and features\n"
	@printf "  make production     Build the optimized release binary with Cargo.lock enforced\n"
	@printf "\n"
	@printf "  make run ARGS=\"status --json\"  Run the CLI through cargo\n"
	@printf "  make clean          Remove Cargo build artifacts\n"

tools:
	@rustc --version
	@$(CARGO) --version
	@$(CARGO) nextest --version
	@git --version

fmt:
	$(CARGO) fmt --all

fmt-check:
	$(CARGO) fmt --all --check

check:
	$(CARGO) check --all-targets --all-features

clippy:
	$(CARGO) clippy --all-targets --all-features -- -D warnings

test:
	$(CARGO) nextest run --all-features

test-doc:
	$(CARGO) test --doc --all-features

build:
	$(CARGO) build --all-targets --all-features

production:
	$(CARGO) build --release --locked --all-features

release: production

prod: production

dev: fmt check clippy test

verify: fmt-check clippy test test-doc build

ci: verify

run:
	$(CARGO) run -- $(ARGS)

clean:
	$(CARGO) clean
