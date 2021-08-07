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
.PHONY: FORCE
SHELL = bash

#
# Test

dlib.cov: test
	test -e $@
	touch $@
test:
	go test -count=1 -coverprofile=dlib.cov -coverpkg=./... -race ./...
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
	GOOS=linux   .circleci/golangci-lint run ./...
	GOOS=darwin  .circleci/golangci-lint run ./...
	GOOS=windows .circleci/golangci-lint run ./...
.PHONY: lint

#
# Tools

.circleci/%: .circleci/%.d/go.mod .circleci/%.d/pin.go
	cd $(<D) && go build -o ../$(@F) $$(sed -En 's,^import "(.*)"$$,\1,p' pin.go)

#
# Utilities for working with borrowed code

GOHOME ?= $(HOME)/src/github.com/golang/go
GOVERSION ?= 1.15.14

%.unmod: % .circleci/goimports FORCE
	<$< \
	  sed \
	    -e '/MODIFIED: META:/d' \
	    -e '/MODIFIED: ADDED/d' \
	    -e 's,.*// MODIFIED: FROM:,,' | \
	  .circleci/goimports -local github.com/datawire/dlib \
	  >$@
borrowed.patch: FORCE
	$(MAKE) $(addsuffix .unmod,$(shell git ls-files ':*borrowed_*'))
	@for copy in $$(git ls-files ':*borrowed_*'); do \
	  orig=$$(sed <<<"$$copy" \
	    -e s,borrowed_,, \
	    -e s,^dexec/internal/,internal/, \
	    -e s,^dexec/,os/exec/, \
	    -e s,^dcontext/,context/, \
	    -e '/^dhttp/{ s,^dhttp/,net/http/internal/,; s,_test,,; }'); \
	  if grep -q 'MODIFIED: META: .* subset ' "$$copy"; then \
	    echo "diff -uw $(GOHOME)/src/$$orig $$copy.unmod | sed '3,\$${ /^-/d; }'" >&2; \
	          diff -uw $(GOHOME)/src/$$orig $$copy.unmod | sed '3,$${ /^-/d; }' || true; \
	  else \
	    echo "diff -uw $(GOHOME)/src/$$orig $$copy.unmod" >&2; \
	          diff -uw $(GOHOME)/src/$$orig $$copy.unmod || true; \
	  fi; \
	done > $@

check-attribution:
	for copy in $$(git ls-files ':*borrowed_*'); do \
	  orig=$$(sed <<<"$$copy" \
	    -e s,borrowed_,, \
	    -e s,^dexec/internal/,internal/, \
	    -e s,^dexec/,os/exec/, \
	    -e s,^dcontext/,context/, \
	    -e '/^dhttp/{ s,^dhttp/,net/http/internal/,; s,_test,,; }'); \
	  if grep -Fq "Go $(GOVERSION) $$orig" "$$copy" && grep -q 'Copyright .* The Go Authors' "$$copy"; then \
	    echo "$$copy : Looks OK"; \
	  else \
	    echo "$$copy : Doesn't claim copied from Go $(GOVERSION) $$orig or doesn't have the copyright statement"; \
	  fi; \
	done
.PHONY: check-attribution
