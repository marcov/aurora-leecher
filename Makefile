APPS := aurora aurora_lambda
LAMBDA_FUNCTION_NAME := aurora
ZIP_FILE := aurora.zip

.PHONY: all
all: $(APPS)

aurora:
	go build -tags "aurora" -o $@

aurora_lambda:
	GOOS="linux" GOARCH="amd64" go build -tags "lambda" -o $@

.PHONY: clean
clean:
	rm -f $(APPS) $(ZIP_FILE)

.PHONY: zip
zip: $(ZIP_FILE)

$(ZIP_FILE): aurora_lambda
	zip -r $@ $^

update-lambda: $(ZIP_FILE)
	aws lambda update-function-code \
		--function-name $(LAMBDA_FUNCTION_NAME) \
		--zip-file fileb://$<
