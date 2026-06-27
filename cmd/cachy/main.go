package main

import (
	"os"

	"github.com/cloud-byte-consulting/cachy/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr, cli.ServeProxy))
}
