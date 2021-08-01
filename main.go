package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go/v3"
)

const (
	APIDomain  = "api.auroranova.it"
	BaseURL    = "https://" + APIDomain + "/"
	configFile = "config.json"
)

var (
	headers = map[string]string{
		"Authority":       APIDomain,
		"Accept":          "application/json, text/plain, */*",
		"Content-Type":    "application/json;charset=UTF-8",
		"Origin":          "https://app.auroranova.it/",
		"Referer":         "https://app.auroranova.it/",
		"Accept-Encoding": "",
		"User-Agent":      "",
	}
)

type httpCredentials struct {
	Username string
	Password string
}

type EmailConfig struct {
	Domain string
	ApiKey string
	From   string
	To     []string
}

type config struct {
	Aurora struct {
		Username   string
		Password   string
		UserId     int
		ActivityId string
	}
	Email EmailConfig
}

func sendEmail(title string, htmlBody string, config *EmailConfig) (string, error) {
	mg := mailgun.NewMailgun(config.Domain, config.ApiKey)
	mg.SetAPIBase(mailgun.APIBaseEU)
	m := mg.NewMessage(
		config.From,
		title,
		"",
		strings.Join(config.To, ", "),
	)

	m.SetHtml(htmlBody)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := mg.Send(ctx, m)
	return id, err
}

func newRequest(method string, url string, reqData []byte, creds *httpCredentials, headers map[string]string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("HTTP new request %v %q failed: %w", method, url, err)
	}

	if creds != nil {
		fmt.Printf("Creds: %s:%s\n", creds.Username, creds.Password)
		req.SetBasicAuth(creds.Username, creds.Password)
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Client Do %v %q failed: %w", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Got invalid HTTP status [%d]: %q", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Resp body ReadAll failed: %w", err)
	}

	return body, nil
}

func main() {
	rawConfig, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Failed to read config file %q: %v\n", configFile, err)
		os.Exit(-1)
	}

	var config config
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		fmt.Printf("Failed to json unmarshal: %v\n", err)
		os.Exit(-1)
	}

	loginData := map[string]interface{}{
		"username": config.Aurora.Username,
		"password": config.Aurora.Password,
	}
	reqRawData, err := json.Marshal(loginData)
	if err != nil {
		fmt.Printf("Failed to json marshal: %v\n", err)
		os.Exit(-1)
	}

	buf, err := newRequest("POST", strings.TrimSuffix(BaseURL, "/")+"/api/login_check", reqRawData, nil, headers)
	if err != nil {
		fmt.Printf("HTTP login request failed: %v\n", err)
		os.Exit(-1)
	}
	//fmt.Printf("Response: %s\n", string(buf))

	var token struct {
		Token string
	}

	if err := json.Unmarshal(buf, &token); err != nil {
		fmt.Printf("JSON unmarshal failed: %v\n", err)
		os.Exit(-1)
	}

	//fmt.Printf("Token: %s\n", token.Token)

	headers["Authorization"] = "Bearer " + token.Token

	noticeFilter := map[string]interface{}{
		"userId":     config.Aurora.UserId,
		"activityId": config.Aurora.ActivityId,
	}
	reqRawData, err = json.Marshal(noticeFilter)
	if err != nil {
		fmt.Printf("Failed to json marshal: %v\n", err)
		os.Exit(-1)
	}

	buf, err = newRequest("GET",
		strings.TrimSuffix(BaseURL, "/")+"/api/app/noticesents?filters="+string(reqRawData),
		nil,
		nil,
		headers)
	if err != nil {
		fmt.Printf("HTTP get notices request failed: %v\n", err)
		os.Exit(-1)
	}
	//fmt.Printf("Response: %s\n", string(buf))

	var httpResp struct {
		Message string `json:"message"`
		Data    []struct {
			Id       int    `json:"id"`
			SendDate string `json:"sendDate"`
			Notice   struct {
				Id          int
				Title       string
				Description string
			} `json:"notice"`
		} `json:"data"`
	}
	//var httpResp interface{}

	if err := json.Unmarshal(buf, &httpResp); err != nil {
		fmt.Printf("JSON unmarshal failed: %v\n", err)
		os.Exit(-1)
	}

	//fmt.Printf("%+v\n", httpResp)

	if len(httpResp.Data) == 0 {
		fmt.Printf("No data in HTTP response, nothing to do: %s\n", buf)
		os.Exit(-1)
	}

	max := 0
	for i, d := range httpResp.Data {
		fmt.Printf("%d - %d - %s - %q\n", d.Id, d.Notice.Id, d.SendDate, d.Notice.Title)
		if httpResp.Data[max].Id < d.Id {
			max = i
		}
	}

	htmlBody := []byte(httpResp.Data[max].Notice.Description)
	//fmt.Printf("%s\n", htmlBody)

	re := regexp.MustCompile(` *style="[^"]*" *`)
	htmlBody = re.ReplaceAll(htmlBody, []byte{})

	re = regexp.MustCompile(` *class="[^"]*" *`)
	htmlBody = re.ReplaceAll(htmlBody, []byte{})

	re = regexp.MustCompile(`<span *> *&nbsp; *</span>`)
	htmlBody = re.ReplaceAll(htmlBody, []byte{})

	//fmt.Printf("\n\n%s\n", htmlBody)

	id := httpResp.Data[max].Id
	title := httpResp.Data[max].Notice.Title
	htmlDoc := fmt.Sprintf(`<html>
<head>
<title>%s</title>
</head>
<body>
<h2>%s</h2>
%s
</body
</html>
`, title, title, htmlBody)

	htmlFileName := fmt.Sprintf("%d.html", id)

	fmt.Printf("Writing notice %d to file %s\n", id, htmlFileName)
	if err := os.WriteFile(htmlFileName, []byte(htmlDoc), 0o664); err != nil {
		fmt.Printf("os WriteFile failed: %v\n", err)
		os.Exit(-1)
	}

	emailID, err := sendEmail(title, htmlDoc, &(config.Email))
	if err != nil {
		fmt.Printf("Email send failed: %v\n", err)
		os.Exit(-1)
	}

	fmt.Printf(">> Send emailID: %v\n", emailID)

	idsToDelete := map[string][]int{
		"id": {id},
	}

	fmt.Printf(">> DEBUG: Not deleting...\n")
	os.Exit(0)

	reqRawData, err = json.Marshal(idsToDelete)
	if err != nil {
		fmt.Printf("Failed to json marshal: %v\n", err)
		os.Exit(-1)
	}

	buf, err = newRequest("PUT", strings.TrimSuffix(BaseURL, "/")+"/api/noticesents/delete", reqRawData, nil, headers)
	if err != nil {
		fmt.Printf("HTTP delete request failed: %v\n", err)
		os.Exit(-1)
	}

	fmt.Printf(">> HTTP req delete response: %s\n", buf)
}
