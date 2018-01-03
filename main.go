package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/iineva/tow/server"
	"github.com/iineva/tow/share"
)

var help = `
  Usage: tow [command] [--help]

  Version: ` + towshare.BuildVersion + `

  Commands:
    server - runs tow in server mode
    client - runs tow in client mode

  Read more:
    https://github.com/iineva/tow

`

func main() {

	flag.Parse()
	args := flag.Args()

	subcmd := ""
	if len(args) > 0 {
		subcmd = args[0]
		args = args[1:]
	}

	switch subcmd {
	case "server":
		server(args)
	case "client":
		client(args)
	default:
		fmt.Fprintf(os.Stderr, help)
		os.Exit(1)
	}
}

func server(args []string) {

	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Print(help)
		os.Exit(1)
	}
	host := flags.String("host", "", "")
	p := flags.String("p", "", "")
	port := flags.String("port", "", "")
	verbose := flags.Bool("v", false, "")

	flags.Parse(args)

	if *host == "" {
		*host = os.Getenv("HOST")
	}
	if *host == "" {
		*host = "0.0.0.0"
	}
	if *port == "" {
		*port = *p
	}
	if *port == "" {
		*port = os.Getenv("PORT")
	}
	if *port == "" {
		*port = "8080"
	}

	s, err := towserver.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	s.Debug = *verbose
	if err = s.Run(fmt.Sprintf("%s:%s", *host, *port)); err != nil {
		log.Fatal(err)
	}
}

func client(args []string) {

}
