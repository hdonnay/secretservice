package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hdonnay/secretservice"
)

var (
	plain = flag.Bool("p", false, "use plain transport instead of encrypted transport")
	l     = log.New(os.Stderr, "getpass\t", log.Ltime)
)

func init() {
	flag.Usage = func() {
		fmt.Println(`getpass [-p] NAME

Prints secret "NAME" string-ified.
`)
		flag.PrintDefaults()
		fmt.Println()
	}
	flag.Parse()
}

func main() {
	algorithm := ss.AlgoDH
	if *plain {
		algorithm = ss.AlgoPlain
	}

	srv, err := ss.DialService()
	if err != nil {
		l.Fatalf("DialService error: %v\n", err)
	}

	session, err := srv.OpenSession(algorithm)
	if err != nil {
		l.Fatalf("OpenSession error: %v\n", err)
	}

	for _, c := range srv.Collections() {
		for _, i := range c.Items() {
			if i.GetLabel() == flag.Arg(0) {
				if i.Locked() {
					// TODO: unlock
					l.Fatalf("item '%s' locked\n", i.GetLabel())
				}
				s, err := i.GetSecret(session)
				if err != nil {
					l.Fatalf("GetSecret error: %v\n", err)
				}
				pass, err := s.GetValue(session)
				if err != nil {
					l.Fatalf("Open error: %v\n", err)
				}
				fmt.Printf("%v", string(pass))
				goto Leave
			}
		}
	}
Leave:
	os.Exit(0)
}
