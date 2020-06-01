export CGO_ENABLED := 0
export NO_PROXY := 127.0.0.1
export GOPROXY := https://goproxy.io,direct

ifeq ($(OS),Windows_NT)
	export HTTP_PROXY := http://127.0.0.1:12769
	export HTTPS_PROXY := http://127.0.0.1:12769
endif

dep:
	go get -u ./... && \
	go fmt ./... && \
	go mod tidy

test:
	go test -v ./...