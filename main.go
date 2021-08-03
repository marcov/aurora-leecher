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

func sendEmail(title string, htmlBody string, attachments map[string][]byte, config *types.EmailConfig) (string, error) {
	mg := mailgun.NewMailgun(config.Domain, config.ApiKey)
	mg.SetAPIBase(mailgun.APIBaseEU)
	m := mg.NewMessage(
		config.From,
		title,
		"",
		strings.Join(config.To, ", "),
	)

	m.SetHtml(htmlBody)

	for name, buf := range attachments {
		log.Printf("email attaching file %q [%d]", name, len(buf))
		m.AddBufferAttachment(name, buf)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := mg.Send(ctx, m)
	return id, err
}

func readConfig() (*config, error) {
	rawConfig, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", configFile, err)
	}

	config := config{}

	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, fmt.Errorf("failed to json unmarshal: %w", err)
	}

	return &config, nil
}

func main() {
	outFile := flag.String("output", "", "Write HTML notice to the specified output file prefix")
	deleteNotices := flag.Bool("delete", false, "Delete notice")
	txEmail := flag.Bool("email", false, "Send email with the notice")
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

	for k, n := range notices {
		log.Printf("Getting notice ID %d", n.Id)
		fatNotice, err := aur.GetNotice(n.Id)
		if err != nil {
			log.Fatalf("Get notice ID %d failed: %v", n.Id, err)
		}
		notice := fatNotice.Notice
		id := fatNotice.Id

		log.Printf("Entry key %s - notice ID %d,%d - %s - %q\n", k, notice.Id, n.Id, n.SendDate, notice.Title)

		var allFiles map[string][]byte
		allFiles, err = aur.GetAllImages(&notice)
		if err != nil {
			log.Fatalf("Get all images failed: %v", err)
		}

		for _, f := range notice.NoticeFiles {
			filename := fmt.Sprintf("%d.%s", f.Id, f.Extension)
			log.Printf("Getting file %q", filename)
			buf, err := aur.GetNoticeFile(f.Id)
			if err != nil {
				log.Printf("Get file %q failed: %v", filename, err)
			} else {
				allFiles[filename] = buf
				log.Printf("Got file %q [%d]", filename, len(buf))
			}
		}

		log.Printf("Generating HTML for notice ID %d", notice.Id)
		htmlDoc, err := aur.GenHtml(fatNotice, nil)
		if err != nil {
			log.Fatalf("Aurora gen HTML failed: %+v", err)
		}

		if *outFile != "" {
			filename := *outFile
			if len(notices) > 1 {
				filename = fmt.Sprintf("%d-%s", id, filename)
			}
			log.Printf("Writing HTML notice ID %d to file %s", id, filename)

			if err := os.WriteFile(filename, []byte(htmlDoc), 0o664); err != nil {
				log.Fatalf("Writing file failed: %v", err)
			}
		}

		if *txEmail {
			log.Printf("Sending e-mail")
			emailID, err := sendEmail(notice.Title, htmlDoc, allFiles, &(config.Email))
			if err != nil {
				log.Fatalf("Email send failed: %v", err)
			}

			log.Printf("Sent email ID: %v", emailID)
		}

		if *deleteNotices {
			log.Printf("Deleting notice ID: %d", fatNotice.Id)

			if err := aur.DeleteIds([]int{fatNotice.Id}); err != nil {
				log.Fatalf("Aurora delete ID failed: %+v", err)
			}
		}
	}
}
