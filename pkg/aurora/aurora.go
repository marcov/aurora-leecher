package aurora

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"localhost/aurora/pkg/types"
)

type Aurora struct {
	Config *types.AuroraConfig
}

type Notice struct {
	Id          int
	Title       string
	Description string
	Gallery     struct {
		ImageName string
	}
	ImageName   string
	LinkVideo   string
	NoticeFiles []struct {
		Id        int
		Label     string
		Extension string
		size      int
	}
}

type NoticeData struct {
	Id       int    `json:"id"`
	SendDate string `json:"sendDate"`
	Notice   Notice `json:"notice"`
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
		return fmt.Errorf("failed to JSON marshal: %w", err)
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

func (au *Aurora) GetNotices() (map[string]NoticeData, error) {
	noticeFilter := map[string]string{
		"userId":     fmt.Sprintf("%d", au.Config.UserId),
		"activityId": fmt.Sprintf("%d", au.Config.ActivityId),
	}

	reqRawData, err := json.Marshal(noticeFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to json marshal: %w", err)
	}

	buf, err := newRequest("GET",
		strings.TrimSuffix(BaseURL, "/")+"/api/app/noticesents?filters="+string(reqRawData),
		nil,
		nil,
		headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP get notices request failed: %w", err)
	}

	//fmt.Printf("Response: %s\n", string(buf))

	var resp struct {
		Message string                `json:"message"`
		Data    map[string]NoticeData `json:"data"`
	}
	//var resp interface{}

	if err := json.Unmarshal(buf, &resp); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	//fmt.Printf("%+v\n", resp)

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no data in HTTP response, nothing to do: %s", buf)
	}

	return resp.Data, nil
}

func (au *Aurora) GetNotice(id int) (*NoticeData, error) {
	buf, err := newRequest("GET",
		fmt.Sprintf("%s/api/app/noticesent/%d", strings.TrimSuffix(BaseURL, "/"), id),
		nil,
		nil,
		headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP get image request failed: %w", err)
	}

	//fmt.Printf("Response: %s\n", string(buf))

	var resp struct {
		Message string     `json:"message"`
		Data    NoticeData `json:"data"`
	}

	if err := json.Unmarshal(buf, &resp); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	//fmt.Printf("%+v\n", resp)

	return &resp.Data, nil
}

func (au *Aurora) DeleteIds(ids []int) error {
	for _, id := range ids {
		idsToDelete := map[string][]int{
			"id": {id},
		}

		reqRawData, err := json.Marshal(idsToDelete)
		if err != nil {
			return fmt.Errorf("failed to JSON marshal: %w", err)
		}

		_, err = newRequest("PUT", strings.TrimSuffix(BaseURL, "/")+"/api/noticesents/delete", reqRawData, nil, headers)
		if err != nil {
			return fmt.Errorf("HTTP delete request failed: %w", err)
		}
	}

	//fmt.Printf(">> HTTP req delete response: %s\n", buf)
	return nil
}

func (au *Aurora) GenHtml(notice *NoticeData, imgs map[string][]byte) (string, error) {
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
`, title, title, htmlBody)

	for name, buf := range imgs {
		htmlDoc += fmt.Sprintf(`<img src="data:image/jpg;base64,%s" alt="img %s">`,
			base64.RawStdEncoding.EncodeToString(buf), name)
	}

	htmlDoc += `
</body>
</html>`

	return htmlDoc, nil
}

func (au *Aurora) GetAllImages(notice *Notice) (map[string][]byte, error) {
	images := make(map[string][]byte)

	if notice.Gallery.ImageName != "" {
		name := notice.Gallery.ImageName
		log.Printf("Getting image %q", name)

		buf, err := au.GetImage("galleries", au.Config.ActivityId, name)
		if err != nil {
			// ignore failures
			log.Printf("Get image %q failed: %v", name, err)
		} else {
			log.Printf("Got image %q [%d]", name, len(buf))
			images[name] = buf
		}
	}

	if notice.ImageName != "" {
		name := notice.ImageName
		log.Printf("Getting image %q", name)

		buf, err := au.GetImage("notices", notice.Id, name)
		if err != nil {
			// ignore failures
			log.Printf("Get image %s failed: %v", name, err)
		} else {
			log.Printf("Got image %q [%d]", name, len(buf))
			images[name] = buf
		}
	}

	return images, nil
}

func (au *Aurora) getFile(path string) ([]byte, error) {
	buf, err := newRequest("GET",
		path,
		nil,
		nil,
		headers)
	if err != nil {
		return nil, fmt.Errorf("HTTP get notices request failed: %w", err)
	}

	return buf, nil
}

func (au *Aurora) GetImage(kind string, id int, name string) ([]byte, error) {
	//https://api.auroranova.it/galleries/images/6/img_5c3c6e618d3a4.jpg
	path := fmt.Sprintf("%s/%s/images/%d/%s", strings.TrimSuffix(BaseURL, "/"), kind, id, name)
	return au.getFile(path)
}

func (au *Aurora) GetNoticeFile(fileId int) ([]byte, error) {
	//https://api.auroranova.it/noticefiles/file/708
	path := fmt.Sprintf("%s/noticefiles/file/%d", strings.TrimSuffix(BaseURL, "/"), fileId)
	return au.getFile(path)
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
		return nil, fmt.Errorf("client Do %v %q failed: %w", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("got invalid HTTP status [%d]: %q", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("resp body ReadAll failed: %w", err)
	}

	return body, nil
}
