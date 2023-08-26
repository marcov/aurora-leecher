APPS := aurora aurora_lambda
LAMBDA_FUNCTION_NAME := aurora
ZIP_FILE := aurora.zip
SRCS := $(shell find . -type f -name "*.go")

.PHONY: all
all: $(APPS)

aurora: $(SRCS)
	go build -tags "aurora" -o $@

# The lambda runtime provided.al2 needs the application to be called "bootstrap"
.PHONY: aurora_lambda
aurora_lambda: bootstrap

# arm64 if it's configured to run on Graviton2
bootstrap: $(SRCS)
	GOOS=linux GOARCH=arm64 go build -tags "lambda" -o $@

.PHONY: clean
clean:
	rm -f aurora bootstrap $(ZIP_FILE)

.PHONY: zip
lambda-zip: $(ZIP_FILE)

$(ZIP_FILE): bootstrap
	zip -r $@ $^

.PHONY: lambda-upload
lambda-upload: $(ZIP_FILE)
	aws lambda update-function-code \
		--function-name $(LAMBDA_FUNCTION_NAME) \
		--zip-file fileb://$<

LAMBDA_TEST_URL := https://22rxq63y4d.execute-api.eu-central-1.amazonaws.com/test/$(LAMBDA_FUNCTION_NAME)
.PHONY: lambda-test
lambda-test:
	curl -v -X POST --data-ascii "{\"txEmail\": true}" $(LAMBDA_TEST_URL)

.PHONY: lambda-invoke
lambda-invoke:
	aws lambda invoke --function-name aurora /dev/stdout --log-type Tail

