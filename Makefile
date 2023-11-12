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
	aws lambda invoke --function-name $(LAMBDA_FUNCTION_NAME) /dev/stdout --log-type Tail

.PHONY: lambda-logs
lambda-logs:
	get_log_stream() { \
		set -x; \
		local -r log_stream_name="$$(aws logs describe-log-streams \
				--log-group-name '/aws/lambda/$(LAMBDA_FUNCTION_NAME)' \
				--query 'logStreams[*].logStreamName' | \
			jq -r '.[-1]'\
		)"; \
		aws logs get-log-events \
			--log-group-name '/aws/lambda/$(LAMBDA_FUNCTION_NAME)' \
			--log-stream-name "$${log_stream_name}" | \
			jq -r '.events | .[].message' | \
			sed '/^$$/d'; \
	}; get_log_stream
