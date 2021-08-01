package main

import (
	"context"
	"encoding/json"
	"fmt"
	"localhost/aurora/pkg/aurora"
	"localhost/aurora/pkg/types"
	"os"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go/v3"
)

const (
	configFile = "config.json"
)

type config struct {
	Aurora types.AuroraConfig
	Email  types.EmailConfig
}

func sendEmail(title string, htmlBody string, config *types.EmailConfig) (string, error) {
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

func readConfig() (*config, error) {
	rawConfig, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file %q: %w", configFile, err)
	}

	config := config{}

	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, fmt.Errorf("Failed to json unmarshal: %w", err)
	}

	return &config, nil
}

func main() {
	config, err := readConfig()
	if err != nil {
		fmt.Printf("Read config failed: %+v\n", err)
		os.Exit(-1)
	}

	aurora := aurora.Aurora{
		Config: &config.Aurora,
	}

	if err := aurora.Login(); err != nil {
		fmt.Printf("Aurora login failed: %+v\n", err)
		os.Exit(-1)
	}

	notices, err := aurora.GetNotices()
	if err != nil {
		fmt.Printf("Aurora get notices failed: %+v\n", err)
		os.Exit(-1)
	}

	max := 0
	for i, d := range notices {
		fmt.Printf("%d - %d - %s - %q\n", d.Id, d.Notice.Id, d.SendDate, d.Notice.Title)
		if notices[max].Id < d.Id {
			max = i
		}
	}
	maxNotice := notices[max]

	htmlDoc, err := aurora.GenHtml(maxNotice)
	if err != nil {
		fmt.Printf("Aurora gen html failed: %+v\n", err)
		os.Exit(-1)
	}

	htmlFileName := fmt.Sprintf("%d.html", maxNotice.Id)
	fmt.Printf("Writing notice %d to file %s\n", maxNotice.Id, htmlFileName)
	if err := os.WriteFile(htmlFileName, []byte(htmlDoc), 0o664); err != nil {
		fmt.Printf("os WriteFile failed: %v\n", err)
		os.Exit(-1)
	}

	emailID, err := sendEmail(maxNotice.Notice.Title, htmlDoc, &(config.Email))
	if err != nil {
		fmt.Printf("Email send failed: %v\n", err)
		os.Exit(-1)
	}

	fmt.Printf(">> Send emailID: %v\n", emailID)

	fmt.Printf(">> DEBUG: Not deleting...\n")
	os.Exit(0)

	aurora.DeleteIds([]int{maxNotice.Id})
	if err != nil {
		fmt.Printf("Aurora delete ID failed: %+v\n", err)
		os.Exit(-1)
	}
}
