DEPS = $(go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)

.PHONY: test deps

test:
	CONFIGOR_ENV=test go test -timeout=30s ./ ./cache/... ./client ./db/... ./raftboltdb ./utils/urlutil

deps:
	go get -d -v ./...
	echo $(DEPS) | xargs -n1 go get -d

