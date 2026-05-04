package domain

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
)

// Result represents the outcome of processing a single URL.
type Result struct {
	XMLName xml.Name `json:"-" xml:"result"`
	URL     string   `json:"url" xml:"url"`
	Title   string   `json:"title" xml:"title"`
	Status  int      `json:"status" xml:"status"`
	Error   string   `json:"error,omitempty" xml:"error,omitempty"`
}

// Exporter defines the interface for exporting results in different formats
type Exporter interface {
	Export(results []Result, filename string) error
}

// JSONExporter implements the Exporter interface for JSON
type JSONExporter struct{}

func (e JSONExporter) Export(results []Result, filename string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// XMLExporter implements the Exporter interface for XML
type XMLExporter struct{}
type XMLResults struct {
	XMLName xml.Name `xml:"results"`
	Items   []Result `xml:"item"`
}

func (e XMLExporter) Export(results []Result, filename string) error {
	data, err := xml.MarshalIndent(XMLResults{Items: results}, "", "  ")
	if err != nil {
		return err
	}
	// XML needs a manual header
	header := []byte(xml.Header)
	return os.WriteFile(filename, append(header, data...), 0644)
}

// TextExporter implements the Exporter interface for classic TXT
type TextExporter struct{}

func (e TextExporter) Export(results []Result, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, r := range results {
		if r.Error != "" {
			fmt.Fprintf(f, "[ERROR] %s: %s\n", r.URL, r.Error)
		} else {
			fmt.Fprintf(f, "[OK %d] %s -> %s\n", r.Status, r.URL, r.Title)
		}
	}
	return nil
}

// ConsoleExporter implements the Exporter interface for console output
type ConsoleExporter struct{}

func (e ConsoleExporter) Export(results []Result, _ string) error {
	fmt.Println("\n--- Results ---")
	for _, r := range results {
		if r.Error != "" {
			fmt.Printf("[ERROR] %s: %s\n", r.URL, r.Error)
		} else {
			fmt.Printf("[OK %d] %s -> %s\n", r.Status, r.URL, r.Title)
		}
	}
	fmt.Println("---------------")

	return nil
}

// GetExporter returns the appropriate Exporter implementation based on the format
func GetExporter(format string) Exporter {
	switch format {
	case "json":
		return JSONExporter{}
	case "xml":
		return XMLExporter{}
	case "txt":
		return TextExporter{}
	default:
		return ConsoleExporter{}
	}
}
