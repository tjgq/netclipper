package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/tjgq/clipboard"
	"github.com/tjgq/netclip"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	defaultKeyFile = ".netclip"
	keyFile        = flag.String("k", "", "path to key file")
	debug          = flag.Bool("d", false, "show debug output")
)

func getKey() ([]byte, error) {

	if *keyFile == "" {
		u, err := user.Current()
		if err != nil {
			return nil, err
		}
		*keyFile = filepath.Join(u.HomeDir, defaultKeyFile)
	}

	data, err := ioutil.ReadFile(*keyFile)
	if err != nil {
		return nil, err
	}

	str := strings.TrimFunc(string(data), unicode.IsSpace)

	return hex.DecodeString(str)
}

func send(p *netclip.Peer, c <-chan string) {
	var last string
	for s := range c {
		if s != last {
			last = s
			err := p.Send(s)
			if *debug {
				switch {
				case err != nil:
					fmt.Fprintf(os.Stderr, "SEND: ERROR: %v\n", err)
				default:
					fmt.Fprintf(os.Stderr, "SEND: %s\n", s)
				}
			}
		}
	}
}

func recv(p *netclip.Peer, c chan<- string) {
	var last string
	for {
		s, _, err := p.Recv()
		if *debug {
			switch {
			case err != nil:
				fmt.Fprintf(os.Stderr, "RECV: ERROR: %v\n", err)
			default:
				fmt.Fprintf(os.Stderr, "RECV: %s\n", s)
			}
		}
		if err == nil && s != last {
			last = s
			c <- s
		}
	}
}

func main() {

	flag.Parse()

	key, err := getKey()
	if err != nil || len(key) != netclip.KeySize {
		fmt.Fprintln(os.Stderr, "No valid key found in .netclip")
		os.Exit(1)
	}

	p := netclip.NewPeer(key)
	if p.Connect() != nil {
		fmt.Fprintln(os.Stderr, "Unable to connect to network")
		os.Exit(1)
	}

	sch := make(chan string, 1)
	rch := make(chan string, 1)

	go send(p, sch)
	go recv(p, rch)

	clipboard.Notify(sch)

	for s := range rch {
		clipboard.Set(s)
	}

	os.Exit(0)
}
