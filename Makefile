.PHONY: build packages linux packages-linux test check pulumi pulumi-test clean
PACKAGES := postgres redis neo4j aiven railway terraform package

build: packages
packages:
	mkdir -p dist
	@for name in $(PACKAGES); do CGO_ENABLED=0 go build -trimpath -o dist/hostlier-$$name ./cmd/hostlier-$$name || exit $$?; done

linux: packages-linux
packages-linux:
	mkdir -p dist
	@for arch in amd64 arm64; do for name in $(PACKAGES); do CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -trimpath -o dist/hostlier-$$name-linux-$$arch ./cmd/hostlier-$$name || exit $$?; done; done

test:
	go test ./...
check:
	go vet ./...
	go test -race ./...

pulumi:
	mkdir -p dist
	cd packages/pulumi && CGO_ENABLED=0 go build -trimpath -o ../../dist/hostlier-pulumi ./cmd/hostlier-pulumi
pulumi-test:
	cd packages/pulumi && go vet ./... && go test -race ./...
clean:
	rm -rf dist
