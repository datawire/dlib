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
SHELL = bash -o pipefail

#
# Tools

# shell scripts
tools/copy-ifchanged = tools/bin/copy-ifchanged
tools/bin/%: tools/src/%.sh
	mkdir -p $(@D)
	install $< $@

# `go get`-able things
tools/gocovmerge    = tools/bin/gocovmerge
tools/goimports     = tools/bin/goimports
tools/golangci-lint = tools/bin/golangci-lint
tools/goveralls     = tools/bin/goveralls
tools/bin/%: tools/src/%/pin.go tools/src/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)

# local Go sources
tools/nodelete = tools/bin/nodelete
tools/bin/.%.stamp: tools/src/%/main.go FORCE
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) .
tools/bin/%: tools/bin/.%.stamp $(tools/copy-ifchanged)
	$(tools/copy-ifchanged) $< $@

#
# Test

dlib.cov: test
	test -e $@
	touch $@
test:
	go test -count=1 -coverprofile=dlib.cov -coverpkg=./... -race ${GOTEST_FLAGS} ./...
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

lint: $(tools/golangci-lint)
	GOOS=linux   $(tools/golangci-lint) run ./...
	GOOS=darwin  $(tools/golangci-lint) run ./...
	GOOS=windows $(tools/golangci-lint) run ./...
.PHONY: lint

#
# Utilities for working with borrowed code

GOHOME ?= $(HOME)/src/github.com/golang/go
GOVERSION ?= 1.15.14

%.unmod: % $(tools/goimports) FORCE
	<$< \
	  sed \
	    -e '/MODIFIED: META:/d' \
	    -e '/MODIFIED: ADDED/d' \
	    -e 's,.*// MODIFIED: FROM:,,' | \
	  $(tools/goimports) -local github.com/datawire/dlib \
	  >$@
borrowed.patch: $(tools/nodelete) FORCE
	$(MAKE) $(addsuffix .unmod,$(shell git ls-files ':*borrowed_*'))
	@for copy in $$(git ls-files ':*borrowed_*'); do \
	  orig=$$(sed <<<"$$copy" \
	    -e s,borrowed_,, \
	    -e s,^dexec/internal/,internal/, \
	    -e s,^dexec/,os/exec/, \
	    -e s,^dcontext/,context/, \
	    -e '/^dhttp/{ s,^dhttp/,net/http/internal/,; s,_test,,; }'); \
	  if grep -q 'MODIFIED: META: .* subset ' "$$copy"; then \
	    echo "{ diff -uw $(GOHOME)/src/$$orig $$copy.unmod || true; } | $(tools/nodelete)" >&2; \
	          { diff -uw $(GOHOME)/src/$$orig $$copy.unmod || true; } | $(tools/nodelete); \
	  else \
	    echo "diff -uw $(GOHOME)/src/$$orig $$copy.unmod || true" >&2; \
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
