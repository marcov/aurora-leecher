APPS := aurora aurora_lambda
LAMBDA_FUNCTION_NAME := aurora
ZIP_FILE := aurora.zip
SRCS := $(shell find . -type f -name "*.go")

.PHONY: all
all: $(APPS)

aurora: $(SRCS)
	go build -tags "aurora" -o $@

aurora_lambda: $(SRCS)
	GOOS="linux" GOARCH="amd64" go build -tags "lambda" -o $@

.PHONY: clean
clean:
	rm -f $(APPS) $(ZIP_FILE)

.PHONY: zip
zip: $(ZIP_FILE)

$(ZIP_FILE): aurora_lambda
	zip -r $@ $^

.PHONY: upload
upload: $(ZIP_FILE)
	aws lambda update-function-code \
		--function-name $(LAMBDA_FUNCTION_NAME) \
		--zip-file fileb://$<

LAMBDA_TEST_URL := https://22rxq63y4d.execute-api.eu-central-1.amazonaws.com/test/aurora

.PHONY: lambda-test
lambda-test:
	curl -v -X POST --data-ascii "{\"txEmail\": true}" $(LAMBDA_TEST_URL)
