fmt:
	gofmt -s -l -w .

test:
	go test ./... $(TESTARGS) -timeout 15m
