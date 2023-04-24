GO = go
GOFLAGS = -ldflags="-s -w"
UPX = upx
UPXFLAGS = --quiet --best --lzma

# The default target:

all: relayer2-tx

.PHONY: all

# Output executables:

relayer2-tx: main.go
	GOARCH=amd64 CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $@ $^ && $(UPX) $(UPXFLAGS) $@

# Rules for development:

clean:
	@rm -Rf relayer2-tx *~

distclean: clean

mostlyclean: clean

maintainer-clean: clean

.PHONY: clean distclean mostlyclean maintainer-clean

.SECONDARY:
.SUFFIXES:
