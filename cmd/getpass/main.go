package main

import (
	"flag"
	"fmt"
	"github.com/hdonnay/secretservice"
	"os"
)

var plain = flag.Bool("p", false, "use plain transport instead of encrypted transport")
var debug = flag.Bool("d", false, "turn on debugging")

func main() {
	flag.Parse()
	algorithm := ss.AlgoDH
	if *plain {
		algorithm = ss.AlgoPlain
	}

	srv, err := ss.DialService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "DialService error: %v\n", err)
		os.Exit(1)
	}
	if *debug {
		fmt.Fprintf(os.Stderr, "debug: service opened\n")
	}

	session, err := srv.OpenSession(algorithm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OpenSession error: %v\n", err)
		os.Exit(1)
	}
	if *debug {
		fmt.Fprintf(os.Stderr, "debug: session opened: %+v\n", session)
	}

	collection, err := srv.Collections()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Collections error: %v\n", err)
		os.Exit(1)
	}
	if *debug {
		fmt.Fprintf(os.Stderr, "debug: collections: %+v\n", collection)
	}

	for _, c := range collection {
		for _, i := range c.Items() {
			if *debug {
				fmt.Fprintf(os.Stderr, "debug: item '%s' %+v\n", i.GetLabel(), i)
			}
			if i.GetLabel() == flag.Arg(0) {
				if i.Locked() {
					fmt.Fprintf(os.Stderr, "item '%s' locked!\n", i.GetLabel())
					os.Exit(1)
				}
				s, err := i.GetSecret(session)
				if err != nil {
					fmt.Fprintf(os.Stderr, "GetSecret error: %v\n%v\n", s, err)
					os.Exit(1)
				}
				pass, err := s.GetSecret(session)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Open error: %v\n%v\n", s, err)
					os.Exit(1)
				}
				fmt.Fprintf(os.Stdout, "%v", string(pass))
				goto Leave
			}
		}
	}
Leave:
	os.Exit(0)
}
