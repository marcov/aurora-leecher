// +build lambda

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

type lambdaEvent struct {
	Body          string   `json:"body"`
	ConfigFile    string   `json:"configFile"`
	OutDir        string   `json:"outDir"`
	DeleteNotices bool     `json:"deleteNotices"`
	TxEmail       bool     `json:"txEmail"`
	Recipients    []string `json:"recipients"`
}

func handleRequest(evt lambdaEvent) (string, error) {
	log.Printf("Raw event: %+v", evt)

	// if it is a HTTP request, unmarshal the body
	if evt.Body != "" {
		// HTTP request, unmarshal the body
		err := json.Unmarshal([]byte(evt.Body), &evt)
		if err != nil {
			err = fmt.Errorf("json unmarshal failed: %w", err)
			return fmt.Sprintf("Aurora failed: %+v", err), err
		}
	}

	err := Run(evt.ConfigFile, evt.OutDir, evt.DeleteNotices, evt.TxEmail, evt.Recipients)
	if err != nil {
		return fmt.Sprintf("Aurora failed: %+v", err), err
	}

	return "Aurora sucess", nil
}

func main() {
	lambda.Start(handleRequest)
}
