#
# Intro

help:
	@echo 'Usage:'
	@echo '  make help'
	@echo '  make test'
	@echo '  make dlib.cov.html'
	@echo '  make lint'
.PHONY: help

.SECONDARY:

#
# Test

dlib.cov: test
test:
	go test -coverprofile=dlib.cov -race ./...
.PHONY: test

%.cov.html: %.cov
	go tool cover -html=$< -o=$@

#
# Generate

generate-clean:
	rm -f dlog/convenience.go
.PHONY: generate-clean

generate:
	go generate ./...
.PHONY: generate

#
# Lint

lint: .circleci/golangci-lint
	.circleci/golangci-lint run ./...
.PHONY: lint

#
# Tools

.circleci/%: .circleci/%.d/go.mod .circleci/%.d/pin.go
	cd $(<D) && go build -o ../$(@F) $$(sed -En 's,^import "(.*)"$$,\1,p' pin.go)
