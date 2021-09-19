// +build !lambda

package main

import (
	"flag"
	"log"
)

func main() {
	configFile := flag.String("config", "", "Config file name")
	outDir := flag.String("output", ".", "Write HTML notices to the specified directory")
	deleteNotices := flag.Bool("delete", false, "Delete notice")
	txEmail := flag.Bool("email", false, "Send email with the notice")
	flag.Parse()

	mode := runMode{
		ConfigFile:    *configFile,
		OutDir:        *outDir,
		DeleteNotices: *deleteNotices,
		TxEmail:       *txEmail,
	}

	if err := Run(mode); err != nil {
		log.Fatalf("%v", err)
	}
}
