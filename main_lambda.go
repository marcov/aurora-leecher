//go:build lambda
// +build lambda

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"net/http"
)

type lambdaEvent struct {
	Body          string   `json:"body"`
	ConfigFile    string   `json:"configFile"`
	OutDir        string   `json:"outDir"`
	DeleteNotices bool     `json:"deleteNotices"`
	TxEmail       bool     `json:"txEmail"`
	Recipients    []string `json:"recipients"`
}

// Add a helper for handling errors. This logs any error to os.Stderr
// and returns a 500 Internal Server Error response that the AWS API
// Gateway understands.
func serverError(err error) (events.APIGatewayProxyResponse, error) {
	log.Printf("Returning error: %+v", err.Error())

	sc := http.StatusInternalServerError
	if err == ErrNoNotices {
		sc = http.StatusNotModified
	}

	return events.APIGatewayProxyResponse{
		StatusCode: sc,
		Body:       fmt.Sprintf("%+v", err),
	}, nil
}

func handleRequest(evt lambdaEvent) (events.APIGatewayProxyResponse, error) {
	log.Print("called handle request")
	log.Printf("Raw event: %+v", evt)

	// if it is a HTTP request, unmarshal the body
	if evt.Body != "" {
		// HTTP request, unmarshal the body
		err := json.Unmarshal([]byte(evt.Body), &evt)
		if err != nil {
			err = fmt.Errorf("json unmarshal failed: %w", err)
			return serverError(err)
		}
	}

	err := Run(evt.ConfigFile, evt.OutDir, evt.DeleteNotices, evt.TxEmail, evt.Recipients)
	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "aurora success",
		//Body:       string(js),
	}, nil
}

func main() {
	fmt.Println("main_lambda.go started!")
	log.Print("calling lambda Start handle request")
	lambda.Start(handleRequest)
}
