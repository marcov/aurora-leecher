package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"io/fs"

	"localhost/aurora/pkg/aurora"
	"localhost/aurora/pkg/config"
	"localhost/aurora/pkg/types"

	"github.com/mailgun/mailgun-go/v3"
)

var (
	ErrNoNotices = fmt.Errorf("no notices found")
)

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

	// remove previously created temp dir if it's there.
	// then, create a new empty one
	auroraTmpDir := "/tmp/aurora"
	_, err := os.Stat(auroraTmpDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat %s: %w", auroraTmpDir, err)
		}
	} else {
		if err := os.RemoveAll(auroraTmpDir); err != nil {
			return "", fmt.Errorf("failed to remove dir %s: %w", auroraTmpDir, err)
		}
	}

	err = os.Mkdir(auroraTmpDir, fs.FileMode(0755))
	if err != nil {
		return "", fmt.Errorf("failed to mkdir %s: %w", auroraTmpDir, err)
	}

	for name, buf := range attachments {
		if strings.HasSuffix(name, "jpg") {
			// Add inline image in a Gmail friendly way, using CID
			filepath := fmt.Sprintf("%s/%s", auroraTmpDir, name)
			if err := os.WriteFile(filepath, buf, fs.FileMode(0644)); err != nil {
				return "", fmt.Errorf("failed to create file %s: %w", filepath, err)
			}
			m.AddInline(filepath)
		} else {
			log.Printf("email attaching file %q [%d]", name, len(buf))
			m.AddBufferAttachment(name, buf)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := mg.Send(ctx, m)
	return id, err
}

func Run(ConfigFile string, OutDir string, DeleteNotices bool, TxEmail bool, Recipients []string) error {
	var err error

	config := config.Config{}
	if ConfigFile == "" {
		log.Print("Reading config from env vars")
		err = config.FromEnvs()
	} else {
		log.Printf("Reading config from file %q", ConfigFile)
		err = config.FromFile(ConfigFile)
	}

	if len(Recipients) > 0 {
		log.Printf("Overriding config email recipients with %+v", Recipients)
		config.Email.To = Recipients
	}

	if err != nil {
		return fmt.Errorf("reading config failed: %w", err)
	}

	if OutDir != "" {
		st, err := os.Stat(OutDir)
		if err != nil {
			return fmt.Errorf("stat failed: %v", err)
		}

		if !st.IsDir() {
			return fmt.Errorf("the specified output %q is not a directory", OutDir)
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
		return ErrNoNotices
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

		log.Printf("Generating HTML for notice ID %d", notice.Id)
		htmlDoc, err := aur.GenHtml(notice, allFiles)
		if err != nil {
			return fmt.Errorf("aurora gen HTML failed: %w", err)
		}

		if OutDir != "" {
			path := filepath.Join(OutDir, fmt.Sprintf("%d.html", id))

			log.Printf("Writing HTML notice to file %q", path)

			if err := os.WriteFile(path, []byte(htmlDoc), 0o664); err != nil {
				return fmt.Errorf("writing file %q failed: %v", path, err)
			}
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

		if TxEmail {
			log.Printf("Sending e-mail")
			emailID, err := sendEmail(notice.Title, htmlDoc, allFiles, &(config.Email))
			if err != nil {
				return fmt.Errorf("email send failed: %v", err)
			}

			log.Printf("Sent email ID: %v", emailID)
		}

		if DeleteNotices {
			log.Printf("Deleting notice ID: %d", id)

			if err := aur.DeleteIds([]int{id}); err != nil {
				return fmt.Errorf("aurora delete ID failed: %w", err)
			}
		}
	}

	return nil
}
