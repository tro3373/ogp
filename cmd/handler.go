package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tro3373/ogp/external/shared"
	"github.com/tro3373/ogp/pkg/ogp"
)

const workers = 2

func handle(args []string) error {
	level, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err == nil {
		log.SetLevel(level)
	}
	log.Debug("Debug start")

	return handleArgs(args)
}

func handleArgs(args []string) error {
	urls := getUrlsFromStdinOrArgs(args)
	if len(urls) == 0 {
		return fmt.Errorf("no url provided")
	}

	apiClient := shared.NewAPIClient(
		shared.WithDumpEnabled(log.GetLevel() >= log.DebugLevel),
	)
	fetcher := ogp.NewFetcher(&apiClientAdapter{client: apiClient})
	results := fetchAll(fetcher, urls)

	log.Debug("Done")
	return printResult(results)
}

func getUrlsFromStdinOrArgs(args []string) []string {
	var urls []string

	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			urls = append(urls, line)
		}
	}
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" {
			continue
		}
		urls = append(urls, trimmed)
	}
	return urls
}

func fetchAll(fetcher *ogp.Fetcher, urls []string) []*ogp.Result {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []*ogp.Result
	)

	sem := make(chan struct{}, workers)

	for _, u := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			log.Debugf("Fetching URL: %s", url)
			result := fetcher.Fetch(url)
			if result.Err != nil {
				log.Warnf("Error fetching %s: %v", url, result.Err)
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(u)
	}

	wg.Wait()
	return results
}

type apiClientAdapter struct {
	client *shared.APIClient
}

func (a *apiClientAdapter) Request(req *http.Request) ([]byte, int, error) {
	return a.client.Request(req)
}

func printResult(results []*ogp.Result) error {
	successful := make([]*ogp.Result, 0, len(results))
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		successful = append(successful, r)
	}

	var target any = successful
	if len(successful) == 1 {
		target = successful[0]
	}

	output, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	fmt.Println(string(output))
	return nil
}
