SSD_VERSION := $(shell git describe --tags 2>/dev/null || echo "v0.0.0")

BINS = snapdown
SSD_SOURCES := $(wildcard */*.go go.mod)

# Colors for output
GREEN = \033[0;32m
NC = \033[0m

all: $(BINS)

clean:
	@echo -e "$(GREEN)Deleting snapdown binary...$(NC)"
	rm -f $(BINS)

.PHONY: all clean local tag tag-minor tag-major releases

snapdown: $(SSD_SOURCES)
	@echo -e "$(GREEN)Building snapdown ${SSD_VERSION} $(NC)"
	go build -o $@ -ldflags "-w -s -X main.VERSION=${SSD_VERSION}"

releases:
	goreleaser release --clean
