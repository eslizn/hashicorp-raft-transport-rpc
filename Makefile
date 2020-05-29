export CGO_ENABLED := 0
export NO_PROXY := 127.0.0.1
export GOPROXY := https://goproxy.io,direct

dep:
	go get -u ./... && \
	go fmt ./... && \
	go mod tidy

test:
	go test -v ./...