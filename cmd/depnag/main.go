package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const mdmclient = "/usr/libexec/mdmclient"

func main() {
	var (
		flDuration = flag.Duration("interval", 25*time.Minute, "interval to check in with DEP")
		flAPIToken = flag.String("server.auth_token", "", "authentication token for depmon server")
		flServer   = flag.String("server.url", "", "url of remote server")
	)
	flag.Parse()
	if *flAPIToken == "" || *flServer == "" {
		flag.Usage()
		os.Exit(1)
	}

	for {
		ticker := time.NewTicker(*flDuration).C
		status := nagDEPAPI()
		if err := reportStatus(*flServer, *flAPIToken, status); err != nil {
			log.Fatal(err)
		}
		log.Printf("last status %s, waiting %s to run again.\n", status, *flDuration)
		<-ticker
	}
}

func reportStatus(serverURL, token, status string) error {
	req, err := http.NewRequest(http.MethodPost, serverURL, strings.NewReader(status))
	if err != nil {
		return fmt.Errorf("creating http request: %s", err)
	}

	req.SetBasicAuth("depmon", token)
	client := &http.Client{Timeout: 15 * time.Second}
	if _, err := client.Do(req); err != nil {
		log.Printf("posting status to server: %s\n", err)
	}
	return nil
}

func nagDEPAPI() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	nagCMD := exec.CommandContext(ctx, mdmclient, "dep", "nag")
	output, err := nagCMD.CombinedOutput()
	if err != nil {
		log.Fatalf("executing dep nag: %s\n", err)
	}
	status := parseCMDCode(output)
	return status
}

// parses Code=12345 and returns either "success" or the error code.
// 34006 = server unavailable
func parseCMDCode(output []byte) string {
	const pattern = `Must run as root|Code=.*\d`
	re := regexp.MustCompile(pattern)
	codeStr := re.Find(output)
	if codeStr == nil {
		return "success"
	}

	// detect if the error is because mdmclient did not run as root.
	if bytes.Equal(codeStr, []byte("Must run as root")) {
		log.Fatal("Must run as root.")
	}

	code := bytes.SplitN(codeStr, []byte("="), 2)[1]
	return string(code)
}
