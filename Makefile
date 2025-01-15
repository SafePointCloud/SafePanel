.PHONY: all safepaneld safepanel-stats safepanel-blocker

all: safepaneld safepanel-stats safepanel-blocker

safepaneld:
	@echo "Building safepaneld..."
	@mkdir -p build
	@go build -o build/safepaneld ./cmd/safepaneld

safepanel-stats:
	@echo "Building safepanel-stats..."
	@mkdir -p build
	@go build -o build/sp-stats ./cmd/safepanel-tui/sp-stats

safepanel-blocker:
	@echo "Building safepanel-blocker..."
	@mkdir -p build
	@go build -o build/sp-blocker ./cmd/safepanel-tui/sp-blocker

restart: safepaneld
	@sudo ./build/safepaneld

stats: safepanel-stats
	@sudo ./build/sp-stats

blocker: safepanel-blocker
	@sudo ./build/sp-blocker
