package aurora

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"localhost/aurora/pkg/types"
)

type Aurora struct {
	Config *types.AuroraConfig
}

type NoticesData struct {
	Id       int    `json:"id"`
	SendDate string `json:"sendDate"`
	Notice   struct {
		Id          int
		Title       string
		Description string
	} `json:"notice"`
}

type httpCredentials struct {
	Username string
	Password string
}

const (
	APIDomain = "api.auroranova.it"
	BaseURL   = "https://" + APIDomain + "/"
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

func (au Aurora) Login() error {
	loginData := map[string]interface{}{
		"username": au.Config.Username,
		"password": au.Config.Password,
	}
	reqRawData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("Failed to JSON marshal: %w", err)
	}

	buf, err := newRequest("POST", strings.TrimSuffix(BaseURL, "/")+"/api/login_check", reqRawData, nil, headers)
	if err != nil {
		return fmt.Errorf("HTTP login request failed: %w", err)
	}
	//fmt.Printf("Response: %s\n", string(buf))

	var token struct {
		Token string
	}

	if err := json.Unmarshal(buf, &token); err != nil {
		return fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	//fmt.Printf("Token: %s\n", token.Token)

	headers["Authorization"] = "Bearer " + token.Token

	return nil
}

func (au *Aurora) GetNotices() ([]NoticesData, error) {
	noticeFilter := map[string]interface{}{
		"userId":     au.Config.UserId,
		"activityId": au.Config.ActivityId,
	}

	reqRawData, err := json.Marshal(noticeFilter)
	if err != nil {
		fmt.Errorf("Failed to json marshal: %w", err)
	}

	buf, err := newRequest("GET",
		strings.TrimSuffix(BaseURL, "/")+"/api/app/noticesents?filters="+string(reqRawData),
		nil,
		nil,
		headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP get notices request failed: %w", err)
	}

	///
	//fmt.Printf("Response: %s\n", string(buf))

	var httpResp struct {
		Message string        `json:"message"`
		Notices []NoticesData `json:"data"`
	}
	//var httpResp interface{}

	if err := json.Unmarshal(buf, &httpResp); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	//fmt.Printf("%+v\n", httpResp)

	if len(httpResp.Notices) == 0 {
		return nil, fmt.Errorf("No data in HTTP response, nothing to do: %s", buf)
	}

	return httpResp.Notices, nil
}

func (au *Aurora) DeleteIds(ids []int) error {
	for _, id := range ids {
		idsToDelete := map[string][]int{
			"id": {id},
		}

		reqRawData, err := json.Marshal(idsToDelete)
		if err != nil {
			return fmt.Errorf("Failed to JSON marshal: %w", err)
		}

		_, err = newRequest("PUT", strings.TrimSuffix(BaseURL, "/")+"/api/noticesents/delete", reqRawData, nil, headers)
		if err != nil {
			return fmt.Errorf("HTTP delete request failed: %w", err)
		}
	}

	//fmt.Printf(">> HTTP req delete response: %s\n", buf)
	return nil
}

func (au *Aurora) GenHtml(notice NoticesData) (string, error) {
	htmlBody := []byte(notice.Notice.Description)
	//fmt.Printf("%s\n", htmlBody)

	re := regexp.MustCompile(` *style="[^"]*" *`)
	htmlBody = re.ReplaceAll(htmlBody, []byte{})

	re = regexp.MustCompile(` *class="[^"]*" *`)
	htmlBody = re.ReplaceAll(htmlBody, []byte{})

	re = regexp.MustCompile(`<span *> *&nbsp; *</span>`)
	htmlBody = re.ReplaceAll(htmlBody, []byte{})

	//fmt.Printf("\n\n%s\n", htmlBody)

	title := notice.Notice.Title
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

	return htmlDoc, nil
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
