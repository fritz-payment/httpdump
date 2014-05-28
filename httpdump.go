package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var errors chan error

// flags
var (
	fullDump                  bool
	readTimeout, writeTimeout time.Duration
)

func main() {
	flag.Usage = help
	flag.BoolVar(&fullDump, "f", false, "whether to dump the raw HTTP request instead of just the body")
	flag.DurationVar(&readTimeout, "rto", 10*time.Second, "HTTP server read timeout")
	flag.DurationVar(&writeTimeout, "wto", 10*time.Second, "HTTP server write timeout")
	flag.Parse()
	addresses := flag.Args()
	if len(addresses) == 0 {
		fmt.Fprint(os.Stderr, "require at least one address to listen to\n\n")
		help()
		os.Exit(1)
	}
	errors = make(chan error)
	go waitOnErrors()
	for _, addr := range addresses {
		srv := &http.Server{
			Addr:         addr,
			Handler:      handler(),
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		}
		go func() {
			err := srv.ListenAndServe()
			if err != nil {
				errors <- err
			}
		}()
	}
	select {}
}

func help() {
	fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "%s [flags] Address [Address...]\n", os.Args[0])
	fmt.Fprint(os.Stderr, "Available flags:\n")
	flag.PrintDefaults()
}

func waitOnErrors() {
	for {
		select {
		case err := <-errors:
			fmt.Fprintf(os.Stderr, "fatal error: %v\n", err)
			os.Exit(1)
		}
	}
}

func handler() http.Handler {
	const outputFormat = `[%s] %s
%s
`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := bytes.NewBuffer(nil)
		if fullDump {
			buf.WriteString(fmt.Sprintf("%s %s %s\n", r.Proto, r.Method, r.RequestURI))
			for k, v := range r.Header {
				buf.WriteString(fmt.Sprintf("%s: %s\n", k, v))
			}
		}
		io.Copy(buf, r.Body)
		fmt.Fprintf(os.Stdout, outputFormat,
			time.Now().Format(time.RFC3339),
			r.URL.String(),
			buf.String())
		r.Body.Close()
	})
}
