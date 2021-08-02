package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"localhost/aurora/pkg/aurora"
	"localhost/aurora/pkg/types"

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
	outFile := flag.String("-output", "", "Write HTML notice to the specified output file")
	deleteNotices := flag.Bool("-delete", false, "Delete notice")
	txEmail := flag.Bool("-email", false, "Send email with the notice")
	flag.Parse()

	log.Printf("Reading config %q", configFile)
	config, err := readConfig()
	if err != nil {
		log.Fatalf("Read config failed: %+v", err)
	}

	aur := aurora.Aurora{
		Config: &config.Aurora,
	}

	log.Printf("Logging in")
	if err := aur.Login(); err != nil {
		log.Fatalf("Aurora login failed: %+v", err)
	}

	log.Printf("Get notices")
	notices, err := aur.GetNotices()
	if err != nil {
		log.Fatalf("Aurora get notices failed: %+v", err)
	}
	log.Printf("Got %d notices", len(notices))

	var maxNotice *aurora.NoticesData
	for _, d := range notices {
		fmt.Printf("%d - %d - %s - %q\n", d.Id, d.Notice.Id, d.SendDate, d.Notice.Title)
		if maxNotice == nil || maxNotice.Id < d.Id {
			maxNotice = &d
		}
	}

	log.Printf("Generating HTML for notice ID: %d", maxNotice.Id)
	htmlDoc, err := aur.GenHtml(maxNotice)
	if err != nil {
		log.Fatalf("Aurora gen HTML failed: %+v", err)
	}

	if *outFile != "" {
		log.Printf("Writing HTML notice ID %d to file %s", maxNotice.Id, *outFile)
		if err := os.WriteFile(*outFile, []byte(htmlDoc), 0o664); err != nil {
			log.Fatalf("os WriteFile failed: %v", err)
		}
	}

	if *txEmail {
		log.Printf("Sending e-mail")
		emailID, err := sendEmail(maxNotice.Notice.Title, htmlDoc, &(config.Email))
		if err != nil {
			log.Fatalf("Email send failed: %v", err)
		}

		log.Printf(">> Sent email ID: %v", emailID)
	}

	if *deleteNotices {
		log.Printf("Deleting notice ID: %d", maxNotice.Id)

		aur.DeleteIds([]int{maxNotice.Id})
		if err != nil {
			log.Fatalf("Aurora delete ID failed: %+v", err)
		}
	}
}
