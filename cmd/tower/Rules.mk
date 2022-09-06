include mk/header.mk
TOW_BIN_$(d) := $(call go-curr-pkg-tgt)
TGT_BIN += $(TOW_BIN_$(d))
TEST_GO_BUILD += $(d)-try-build
CLEAN += $(TOW_BIN_$(d))

PATH := $(realpath $(d)):$(PATH)

$(d)-try-build $(TOW_BIN_$(d)): GOFLAGS += $(cmd/tower_flags)

$(TOW_BIN_$(d)): $(d) $$(DEPS_GO) ALWAYS #| $(DEPS_OO_$(d))
	$(go-build-relative)

TRY_BUILD_$(d)=$(addprefix $(d)-try-build-,$(SUPPORTED_PLATFORMS))
$(d)-try-build: $(TRY_BUILD_$(d))
.PHONY: $(d)-try-build

include mk/footer.mk