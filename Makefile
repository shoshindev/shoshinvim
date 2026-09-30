.PHONY: build test check run cross screenshots notices
build:
	go build -trimpath -o bin/shoshinvim ./cmd/shoshinvim
test:
	go test -race ./...
check:
	go vet ./...
	go test ./...
run:
	go run ./cmd/shoshinvim
cross:
	@mkdir -p bin
	@for target in darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64; do \
		os=$${target%/*}; arch=$${target#*/}; suffix=; [ "$$os" != windows ] || suffix=.exe; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -trimpath -o bin/shoshinvim-$$os-$$arch$$suffix ./cmd/shoshinvim || exit; \
	done
screenshots:
	SHOSHINVIM_SCREENSHOTS=docs/assets go test ./internal/ui -run TestScreenshots -count=1
notices:
	python3 scripts/third-party-notices.py
