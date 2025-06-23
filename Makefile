SSD_VERSION := $(shell git describe --tags 2>/dev/null || echo "v0.0.0")

BINS = snapsnapdown
SSD_SOURCES := $(wildcard */*.go go.mod)

# Colors for output
GREEN = \033[0;32m
NC = \033[0m

all: $(BINS)

clean:
	@echo -e "$(GREEN)Deleting snapsnapdown binary...$(NC)"
	rm -f $(BINS)

.PHONY: all clean local tag tag-minor tag-major releases

snapsnapdown: $(SSD_SOURCES)
	@echo -e "$(GREEN)Building snapsnapdown ${SSD_VERSION} $(NC)"
	go build -o $@ -ldflags "-w -s -X main.VERSION=${SSD_VERSION}"

releases:
	goreleaser release --clean
