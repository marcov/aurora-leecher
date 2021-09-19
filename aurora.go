package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"localhost/aurora/pkg/aurora"
	"localhost/aurora/pkg/config"
	"localhost/aurora/pkg/types"

	"github.com/mailgun/mailgun-go/v3"
)

type runMode struct {
	ConfigFile    string `json:"configFile"`
	OutDir        string `json:"outDir"`
	DeleteNotices bool   `json:"deleteNotices"`
	TxEmail       bool   `json:"txEmail"`
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

func Run(mode runMode) error {
	var err error

	log.Printf("Run mode: %+v", mode)

	config := config.Config{}
	if mode.ConfigFile == "" {
		log.Print("Reading config from env vars")
		err = config.FromEnvs()
	} else {
		log.Printf("Reading config from file %q", mode.ConfigFile)
		err = config.FromFile(mode.ConfigFile)
	}

	if err != nil {
		return fmt.Errorf("reading config failed: %w", err)
	}

	if mode.OutDir != "" {
		st, err := os.Stat(mode.OutDir)
		if err != nil {
			return fmt.Errorf("stat failed: %v", err)
		}

		if !st.IsDir() {
			return fmt.Errorf("the specified output %q is not a directory", mode.OutDir)
		}
	}

	aur := aurora.Aurora{
		Config: &config.Aurora,
	}

	log.Printf("Logging in")
	if err := aur.Login(); err != nil {
		return fmt.Errorf("aurora login failed: %w", err)
	}

	log.Printf("Checking notices")
	ids, err := aur.CheckNotices()
	if err != nil {
		return fmt.Errorf("check notices failed: %w", err)
	}

	if len(ids) == 0 {
		log.Printf("No notices found, nothing to do, exiting")
		return nil
	}

	log.Printf("Found %d notices", len(ids))

	for _, id := range ids {
		log.Printf("Getting notice ID %d", id)
		retId, notice, err := aur.GetNotice(id)
		if err != nil {
			return fmt.Errorf("get notice failed: %v", err)
		}

		if id != retId {
			return fmt.Errorf("expected ID %d but found %d", id, retId)
		}

		var allFiles map[string][]byte
		allFiles, err = aur.GetAllImages(notice)
		if err != nil {
			return fmt.Errorf("get all images failed: %v", err)
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
			return fmt.Errorf("aurora gen HTML failed: %w", err)
		}

		if mode.OutDir != "" {
			path := filepath.Join(mode.OutDir, fmt.Sprintf("%d.html", id))

			log.Printf("Writing HTML notice to file %q", path)

			if err := os.WriteFile(path, []byte(htmlDoc), 0o664); err != nil {
				return fmt.Errorf("writing file %q failed: %v", path, err)
			}
		}

		if mode.TxEmail {
			log.Printf("Sending e-mail")
			emailID, err := sendEmail(notice.Title, htmlDoc, allFiles, &(config.Email))
			if err != nil {
				return fmt.Errorf("email send failed: %v", err)
			}

			log.Printf("Sent email ID: %v", emailID)
		}

		if mode.DeleteNotices {
			log.Printf("Deleting notice ID: %d", id)

			if err := aur.DeleteIds([]int{id}); err != nil {
				return fmt.Errorf("aurora delete ID failed: %w", err)
			}
		}
	}

	return nil
}
