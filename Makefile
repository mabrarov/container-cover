# https://clarkgrubb.com/makefile-style-guide
MAKEFLAGS     += --warn-undefined-variables
SHELL         := bash
.SHELLFLAGS   := -eu -o pipefail -c
.DEFAULT_GOAL := all
.DELETE_ON_ERROR:
.SUFFIXES:

GO                 ?= go
GOFUMPT            ?= gofumpt
GOFUMPT_VERSION    ?= v0.9.2
GOCOVMERGE         ?= gocovmerge
GOCOVMERGE_VERSION ?= v2.14.0
MAKEFILE_DIR       := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
PREFIX             ?= $(MAKEFILE_DIR)/.build
BINNAME            ?= app
COVER_INSTRUMENT   ?= 0

# https://tensin.name/blog/makefile-escaping.html
define noexpand
    ifeq ($$(origin $(1)),environment)
        $(1) := $$(value $(1))
    else ifeq ($$(origin $(1)),environment override)
        $(1) := $$(value $(1))
    else ifeq ($$(origin $(1)),command line)
        override $(1) := $$(value $(1))
    endif
endef

$(eval $(call noexpand,GO))
$(eval $(call noexpand,GOFUMPT))
$(eval $(call noexpand,GOFUMPT_VERSION))
$(eval $(call noexpand,GOCOVMERGE))
$(eval $(call noexpand,GOCOVMERGE_VERSION))
$(eval $(call noexpand,PREFIX))
$(eval $(call noexpand,BINNAME))
$(eval $(call noexpand,COVER_INSTRUMENT))

escape = $(subst ','\'',$(1))
squote = '$(call escape,$(1))'

APP_BUILD_OUTPUT       := $(PREFIX)/$(BINNAME)$(shell $(call squote,$(GO)) env GOEXE)
COVER_OUTPUT           := $(MAKEFILE_DIR)/coverage.out
COVER_OUTPUT_HOST      := $(MAKEFILE_DIR)/.build/coverage/host/coverage.out
COVER_OUTPUT_CONTAINER := $(MAKEFILE_DIR)/.build/coverage/container/coverage.out
COVER_CONTAINER_DIR    := //tmp/cover
COVER_HOST_DIR         := $(MAKEFILE_DIR)/.build/coverage/container/cover

ifneq ($(COVER_INSTRUMENT),0)
    COVER_BUILD_OPTIONS := -cover -covermode count
else
    COVER_BUILD_OPTIONS :=
endif

.PHONY: all
all: format test build

.PHONY: deps
deps: format-deps test-cover-deps

.PHONY: format
format: format-deps
	cd $(call squote,$(MAKEFILE_DIR)) && $(call squote,$(GOFUMPT)) -l -w .

.PHONY: format-deps
format-deps:
	$(call squote,$(GO)) install mvdan.cc/gofumpt@$(call squote,$(GOFUMPT_VERSION))

.PHONY: test-short
test-short:
	$(call squote,$(GO)) test -C $(call squote,$(MAKEFILE_DIR)) -short -v -count 1 ./...

.PHONY: test
test:
	$(call squote,$(GO)) test -C $(call squote,$(MAKEFILE_DIR)) -v -count 1 ./...

.PHONY: test-cover
test-cover: test-cover-deps
	$(RM) -r $(call squote,$(COVER_HOST_DIR))
	mkdir -p "$$(dirname $(call squote,$(COVER_OUTPUT_HOST)))"
	COVER_CONTAINER_DIR=$(call squote,$(COVER_CONTAINER_DIR)) COVER_HOST_DIR=$(call squote,$(COVER_HOST_DIR)) $(call squote,$(GO)) test -C $(call squote,$(MAKEFILE_DIR)) -v -count 1 -cover -covermode count -coverpkg=./... -coverprofile=$(call squote,$(COVER_OUTPUT_HOST)) ./...
	$(call squote,$(GO)) tool -C $(call squote,$(MAKEFILE_DIR)) covdata textfmt -i=$(call squote,$(COVER_HOST_DIR)) -o $(call squote,$(COVER_OUTPUT_CONTAINER))
	$(call squote,$(GOCOVMERGE)) -o $(call squote,$(COVER_OUTPUT)) $(call squote,$(COVER_OUTPUT_HOST)) $(call squote,$(COVER_OUTPUT_CONTAINER))
	$(call squote,$(GO)) tool -C $(call squote,$(MAKEFILE_DIR)) cover -func=$(call squote,$(COVER_OUTPUT))

.PHONY: test-cover-deps
test-cover-deps:
	$(call squote,$(GO)) install github.com/alexfalkowski/gocovmerge/v2@$(call squote,$(GOCOVMERGE_VERSION))

.PHONY: build
build:
	CGO_ENABLED=0 $(call squote,$(GO)) build -C $(call squote,$(MAKEFILE_DIR))$(if $(COVER_BUILD_OPTIONS), $(COVER_BUILD_OPTIONS),) -trimpath -o $(call squote,$(APP_BUILD_OUTPUT)) ./cmd/app

.PHONY: clean
clean: clean-app clean-coverage

.PHONY: clean-app
clean-app:
	$(RM) $(call squote,$(APP_BUILD_OUTPUT))

.PHONY: clean-coverage
clean-coverage:
	$(RM) $(call squote,$(COVER_OUTPUT))
	$(RM) $(call squote,$(COVER_OUTPUT_HOST))
	$(RM) $(call squote,$(COVER_OUTPUT_CONTAINER))
	$(RM) -r $(call squote,$(COVER_HOST_DIR))
