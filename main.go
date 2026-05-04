package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopher-spy/internal/domain"

	"github.com/PuerkitoBio/goquery"
	"github.com/schollz/progressbar/v3"
)

func main() {
	config, err := parseAppParameters()
	if err != nil {
		os.Exit(1)
	}

	processUrls(*config)
}

// parseAppParameters handles the command-line arguments and flags, returning an
// AppConfig struct
func parseAppParameters() (*domain.AppConfig, error) {
	format := flag.String("format", "console", "Output format: console, txt, json, xml")
	rate := flag.Int("rate", 2, "Requests per second")
	inputFile := flag.String("input", "", "Text file containing a list of URLs (one per line)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of GopherSpy:\n")
		fmt.Fprintf(os.Stderr, "  go run main.go [flags] url1 url2 url3...\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	var urls []string

	if *inputFile != "" {
		fileUrls, err := readUrlsFromFile(*inputFile)
		if err != nil {
			fmt.Printf("Error reading input file: %v\n", err)
			return nil, err
		}
		urls = append(urls, fileUrls...)
	} else {
		urls = append(urls, flag.Args()...)
	}

	if len(urls) == 0 {
		fmt.Println("Error: No URLs provided.")
		flag.Usage()
		return nil, fmt.Errorf("No URLs provided")
	}

	config := domain.AppConfig{
		Format:    *format,
		Rate:      *rate,
		InputFile: *inputFile,
		URLs:      urls,
	}

	return &config, nil
}

// readUrlsFromFile reads a file and extracts URLs line by line
func readUrlsFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines, scanner.Err()
}

// processUrls groups the concurrency logic we have built so far, making it easier to
// read and maintain
func processUrls(config domain.AppConfig) {
	// The Ticker sends a signal on a channel at regular intervals
	ticker := time.NewTicker(time.Second / time.Duration(config.Rate))
	defer ticker.Stop()

	resultsChan := make(chan domain.Result, len(config.URLs))
	var wg sync.WaitGroup

	fmt.Printf("Starting processing of %d URLs with a limit of %d req/s...\n\n", len(config.URLs), config.Rate)

	bar := progressbar.Default(int64(len(config.URLs)), "Processing URLs...")

	// Set graceful shutdown on Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n[!] Program interrupted. Shutting down gracefully...")
		cancel()
	}()

	counter := 0

loop:
	for _, url := range config.URLs {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			// Wait for the next "tick" of the clock to respect the Rate Limit
			wg.Add(1)
			go func(u string) {
				defer wg.Done()
				resultsChan <- fetchTitle(u)
				counter++
				bar.Add(1)
			}(url)
		}
	}

	// Goroutine to close the channel once all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	exportResults(resultsChan, config.Format)

	fmt.Printf("\n%d out of %d URLs processed.\n", counter, len(config.URLs))
}

// fetchTitle performs the HTTP request and extracts the page title, returning a
// Result struct with the outcome
func fetchTitle(url string) domain.Result {
	client := http.Client{Timeout: 5 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return domain.Result{URL: url, Error: err.Error()}
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return domain.Result{URL: url, Status: res.StatusCode, Error: err.Error()}
	}

	return domain.Result{
		URL:    url,
		Title:  doc.Find("title").Text(),
		Status: res.StatusCode,
	}
}

// exportResults decides where to print the data
func exportResults(results <-chan domain.Result, format string) {
	var exporter domain.Exporter
	var resultsSlice = make([]domain.Result, 0)

	for res := range results {
		resultsSlice = append(resultsSlice, res)
	}

	filename := fmt.Sprintf("results.%s", format)

	exporter = domain.GetExporter(format)

	err := exporter.Export(resultsSlice, filename)
	if err != nil {
		fmt.Printf("\nError exporting results: %v\n", err)
	} else {
		if format != "console" {
			fmt.Printf("\nResults will be saved in '%s'.\n", filename)
		}
	}
}
