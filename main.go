// +build !lambda

package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	configFile := flag.String("config", "", "Config file name")
	outDir := flag.String("output", ".", "Write HTML notices to the specified directory")
	deleteNotices := flag.Bool("delete", false, "Delete notice")
	txEmail := flag.Bool("email", false, "Send email with the notice")
	flag.Parse()

	err := Run(*configFile, *outDir, *deleteNotices, *txEmail, nil)
	if err == nil {
		return
	}

	log.Printf("%v", err)
	if err != ErrNoNotices {
		os.Exit(1)
	}
}
