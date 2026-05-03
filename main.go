package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/schollz/progressbar/v3"
)

// Result represents the outcome of processing a single URL, including the URL
// itself, the page title, HTTP status code, and any error encountered during the
// process.
type Result struct {
	URL    string
	Title  string
	Status int
	Error  error
}

// AppConfig groups the program's configuration settings to facilitate parameter
// passing
type AppConfig struct {
	ToFile    bool
	Rate      int
	InputFile string
	URLs      []string
}

func main() {
	// Parse command-line parameters and validate them, exiting with an error if
	// something is wrong with the input
	config, err := parseAppParameters()
	if err != nil {
		os.Exit(1)
	}

	// Start processing the URLs based on the provided configuration
	processUrls(*config)
}

func parseAppParameters() (*AppConfig, error) {
	// Define flags
	toFile := flag.Bool("file", false, "Write the result in a file 'results.txt'")
	rate := flag.Int("rate", 2, "Requests per second")
	inputFile := flag.String("input", "", "Text file containing a list of URLs (one per line)")

	// Customize message to show when the user runs the program with -h or --help
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of GopherSpy:\n")
		fmt.Fprintf(os.Stderr, "  go run main.go [flags] url1 url2 url3...\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	var urls []string

	if *inputFile != "" {
		// Add URLs from a file if the -input flag is used
		fileUrls, err := readUrlsFromFile(*inputFile)
		if err != nil {
			fmt.Printf("Error reading input file: %v\n", err)
			return nil, err
		}
		urls = append(urls, fileUrls...)
	} else {
		// Add URLs passed directly in the command (separated by space)
		urls = append(urls, flag.Args()...)
	}

	// Validation: If no URLs are provided, stop execution and show an error message
	if len(urls) == 0 {
		fmt.Println("Error: No URLs provided.")
		flag.Usage()
		return nil, fmt.Errorf("No URLs provided")
	}

	config := AppConfig{
		ToFile:    *toFile,
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
func processUrls(config AppConfig) {
	// The Ticker sends a signal on a channel at regular intervals
	ticker := time.NewTicker(time.Second / time.Duration(config.Rate))
	defer ticker.Stop()

	resultsChan := make(chan Result, len(config.URLs))
	var wg sync.WaitGroup

	fmt.Printf("Starting processing of %d URLs with a limit of %d req/s...\n\n", len(config.URLs), config.Rate)

	bar := progressbar.Default(int64(len(config.URLs)), "Processing URLs...")

	for _, url := range config.URLs {
		wg.Add(1)

		// Wait for the next "tick" of the clock to respect the Rate Limit
		<-ticker.C

		go func(u string) {
			defer wg.Done()
			resultsChan <- fetchTitle(u)
			bar.Add(1)
		}(url)
	}

	// Goroutine to close the channel once all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Export results as they come in, either to console or to a file
	exportResults(resultsChan, config.ToFile)

	fmt.Println("\nAll URLs processed.")
	if config.ToFile {
		fmt.Println("Results will be saved in 'results.txt'.")
	}
}

// fetchTitle performs the HTTP request and extracts the page title, returning a
// Result struct with the outcome
func fetchTitle(url string) Result {
	res, err := http.Get(url)
	if err != nil {
		return Result{URL: url, Error: err}
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return Result{URL: url, Status: res.StatusCode, Error: err}
	}

	return Result{
		URL:    url,
		Title:  doc.Find("title").Text(),
		Status: res.StatusCode,
	}
}

// exportResults decides where to print the data
func exportResults(results <-chan Result, toFile bool) {
	var f *os.File
	var err error

	if toFile {
		f, err = os.Create("results.txt")
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer f.Close()
	}

	for res := range results {
		output := ""
		if res.Error != nil {
			output = fmt.Sprintf("[ERROR] %s: %v\n", res.URL, res.Error)
		} else {
			output = fmt.Sprintf("[OK %d] %s -> %s\n", res.Status, res.URL, res.Title)
		}

		if toFile {
			f.WriteString(output)
		} else {
			fmt.Print(output)
		}
	}
}
