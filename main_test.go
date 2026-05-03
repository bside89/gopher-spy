package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainFlow_ValidURLs_PrintToConsole(t *testing.T) {
	server := newTestServer()
	output := runApplication(t, []string{"-rate", "1000", server.URL})

	if !strings.Contains(output, "Starting processing of 1 URLs with a limit of 1000 req/s") {
		t.Fatalf("expected start message in output, got:\n%s", output)
	}

	if !strings.Contains(output, fmt.Sprintf("[OK 200] %s -> Gopher Spy", server.URL)) {
		t.Fatalf("expected success line in output, got:\n%s", output)
	}
}

func TestMainFlow_InvalidURL_PrintToConsole(t *testing.T) {
	server := newTestServer()
	output := runApplication(t, []string{"-rate", "1000", server.URL, "http://%"})

	if !strings.Contains(output, fmt.Sprintf("[OK 200] %s -> Gopher Spy", server.URL)) {
		t.Fatalf("expected success line in output, got:\n%s", output)
	}

	if !strings.Contains(output, "[ERROR] http://%:") {
		t.Fatalf("expected error line in output, got:\n%s", output)
	}
}

func TestMainFlow_ValidURLs_SaveToFile(t *testing.T) {
	server := newTestServer()
	workDir := t.TempDir()
	chdir(t, workDir)

	output := runApplication(t, []string{"-rate", "1000", "-file", server.URL})
	if !strings.Contains(output, "Results will be saved in 'results.txt'...") {
		t.Fatalf("expected file output notice, got:\n%s", output)
	}

	content := readFile(t, filepath.Join(workDir, "results.txt"))
	if !strings.Contains(content, fmt.Sprintf("[OK 200] %s -> Gopher Spy", server.URL)) {
		t.Fatalf("expected success line in results.txt, got:\n%s", content)
	}
}

func TestMainFlow_InvalidURL_SaveToFile(t *testing.T) {
	server := newTestServer()
	workDir := t.TempDir()
	chdir(t, workDir)

	runApplication(t, []string{"-rate", "1000", "-file", server.URL, "http://%"})

	content := readFile(t, filepath.Join(workDir, "results.txt"))
	if !strings.Contains(content, fmt.Sprintf("[OK 200] %s -> Gopher Spy", server.URL)) {
		t.Fatalf("expected success line in results.txt, got:\n%s", content)
	}

	if !strings.Contains(content, "[ERROR] http://%:") {
		t.Fatalf("expected error line in results.txt, got:\n%s", content)
	}
}

func TestMainFlow_InputFile_ValidURLs_PrintToConsole(t *testing.T) {
	server := newTestServer()
	inputFile := writeInputFile(t, []string{server.URL})
	output := runApplication(t, []string{"-rate", "1000", "-input", inputFile})

	if !strings.Contains(output, fmt.Sprintf("[OK 200] %s -> Gopher Spy", server.URL)) {
		t.Fatalf("expected success line in output, got:\n%s", output)
	}
}

func TestMainFlow_InputFile_InvalidURLs_PrintToConsole(t *testing.T) {
	server := newTestServer()
	inputFile := writeInputFile(t, []string{server.URL, "http://%"})
	output := runApplication(t, []string{"-rate", "1000", "-input", inputFile})

	if !strings.Contains(output, fmt.Sprintf("[OK 200] %s -> Gopher Spy", server.URL)) {
		t.Fatalf("expected success line in output, got:\n%s", output)
	}

	if !strings.Contains(output, "[ERROR] http://%:") {
		t.Fatalf("expected error line in output, got:\n%s", output)
	}
}

func runApplication(t *testing.T, args []string) string {
	t.Helper()

	originalArgs := os.Args
	originalCommandLine := flag.CommandLine
	os.Args = append([]string{"gopher-spy"}, args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	t.Cleanup(func() {
		os.Args = originalArgs
		flag.CommandLine = originalCommandLine
	})

	config, err := parseAppParameters()
	if err != nil {
		t.Fatalf("parseAppParameters() returned error: %v", err)
	}

	return captureOutput(func() {
		processUrls(*config)
	})
}

func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	os.Stdout = w
	outputChan := make(chan string)

	go func() {
		var buffer bytes.Buffer
		_, _ = io.Copy(&buffer, r)
		outputChan <- buffer.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	return <-outputChan
}

func newTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, "<html><head><title>Gopher Spy</title></head><body></body></html>")
	}))
}

func writeInputFile(t *testing.T, urls []string) string {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), "input.txt")
	content := strings.Join(urls, "\n") + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	return filePath
}

func readFile(t *testing.T, filePath string) string {
	t.Helper()

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", filePath, err)
	}

	return string(data)
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}
