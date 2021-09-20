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

	if err := Run(*configFile, *outDir, *deleteNotices, *txEmail, nil); err != nil {
		log.Fatalf("%v", err)
	}
}
