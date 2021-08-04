package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"localhost/aurora/pkg/aurora"
	"localhost/aurora/pkg/types"

	"github.com/mailgun/mailgun-go/v3"
)

const (
	defaultConfigFile = "config.json"
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

func readConfig(configFile string) (*config, error) {
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
	configFile := flag.String("config", defaultConfigFile, "Config file name")
	outDir := flag.String("output", ".", "Write HTML notices to the specified directory")
	deleteNotices := flag.Bool("delete", false, "Delete notice")
	txEmail := flag.Bool("email", false, "Send email with the notice")
	flag.Parse()

	log.Printf("Reading config %q", *configFile)
	config, err := readConfig(*configFile)
	if err != nil {
		log.Fatalf("Read config failed: %+v", err)
	}

	if *outDir != "" {
		st, err := os.Stat(*outDir)
		if err != nil {
			log.Fatalf("stat failed: %v", err)
		}

		if !st.IsDir() {
			log.Fatalf("The specified output %q is not a directory", *outDir)
		}
	}

	aur := aurora.Aurora{
		Config: &config.Aurora,
	}

	log.Printf("Logging in")
	if err := aur.Login(); err != nil {
		log.Fatalf("Aurora login failed: %+v", err)
	}

	log.Printf("Checking notices")
	ids, err := aur.CheckNotices()
	if err != nil {
		log.Fatalf("Check notices failed: %+v", err)
	}

	if len(ids) == 0 {
		log.Printf("No notices found, nothing to do, exiting")
		os.Exit(0)
	}

	log.Printf("Found %d notices", len(ids))

	for _, id := range ids {
		log.Printf("Getting notice ID %d", id)
		retId, notice, err := aur.GetNotice(id)
		if err != nil {
			log.Fatalf("Get notice failed: %v", err)
		}

		if id != retId {
			log.Fatalf("Expected ID %d but found %d", id, retId)
		}

		var allFiles map[string][]byte
		allFiles, err = aur.GetAllImages(notice)
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
		htmlDoc, err := aur.GenHtml(notice, nil)
		if err != nil {
			log.Fatalf("Aurora gen HTML failed: %+v", err)
		}

		if *outDir != "" {
			path := filepath.Join(*outDir, fmt.Sprintf("%d.html", id))

			log.Printf("Writing HTML notice to file %q", path)

			if err := os.WriteFile(path, []byte(htmlDoc), 0o664); err != nil {
				log.Fatalf("Writing file %q failed: %v", path, err)
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
			log.Printf("Deleting notice ID: %d", id)

			if err := aur.DeleteIds([]int{id}); err != nil {
				log.Fatalf("Aurora delete ID failed: %+v", err)
			}
		}
	}
}
