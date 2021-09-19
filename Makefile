APPS := aurora aurora_lambda

all: $(APPS)

.PHONY: aurora
aurora:
	go build -tags "aurora" -o $@

.PHONY: aurora_lambda
aurora_lambda:
	GOOS="linux" GOARCH="amd64" go build -tags "lambda" -o $@

.PHONY: clean
clean:
	rm -f $(APPS)

.PHONY: zip
zip: aurora_lambda
	zip -r aurora.zip $^
