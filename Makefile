help:
	@echo 'Usage:'
	@echo '  make help'
	@echo '  make test'
	@echo '  make dlib.cov.html'
	@echo '  make lint'
.PHONY: help

dlib.cov: test
test:
	go test -coverprofile=dlib.cov -race ./...
.PHONY: test

%.cov.html: %.cov
	go tool cover -html=$< -o=$@

.circleci/%: .circleci/%.d/go.mod .circleci/%.d/pin.go
	cd $(<D) && go build -o ../$(@F) $$(sed -En 's,^import "(.*)"$$,\1,p' pin.go)

lint: .circleci/golangci-lint
	.circleci/golangci-lint run ./...
.PHONY: lint

.SECONDARY:
