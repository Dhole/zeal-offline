package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const urlFeeds = "https://github.com/Kapeli/feeds/archive/refs/heads/master.zip"
const outPath = "feeds"

type Entry struct {
	Version string   `xml:"version"`
	Urls    []string `xml:"url"`
}

func run() error {
	if err := os.MkdirAll(outPath, 0o755); err != nil {
		return fmt.Errorf("Can't create %v dir: %w", outPath, err)
	}
	req, err := http.NewRequest("GET", urlFeeds, nil)
	if err != nil {
		return fmt.Errorf("Can't create http request to %v: %w", urlFeeds, err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Can't do http request to %v: %w", urlFeeds, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Can't read http body at %v: %w", urlFeeds, err)
	}

	r, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("Can't create zip reader: %w", err)
	}
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		base := strings.TrimSuffix(filepath.Base(f.Name), ".xml")
		log.Printf("+ %s", base)
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("Can't open %v: %w", f.Name, err)
		}
		content, err := io.ReadAll(rc)
		if err != nil {
			return fmt.Errorf("Can't read %v: %w", f.Name, err)
		}
		rc.Close()
		// log.Printf("%v\n", string(content))
		var entry Entry
		if err := xml.Unmarshal(content, &entry); err != nil {
			return fmt.Errorf("Can't xml.Unmarshal entry %v: %w", f.Name, err)
		}
		// log.Printf("%+v\n", entry)
		for _, url := range entry.Urls {
			parts := strings.Split(url, "/")
			filename := parts[len(parts)-1]
			filePath := filepath.Join(outPath, filename)
			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("Can't create file %v: %w", filePath, err)
			}
			log.Printf("Fetching %v ...", url)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("Can't create http request to %v: %w", url, err)
			}
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("Can't do http request to %v: %w", url, err)
			}
			defer resp.Body.Close()
			if _, err := io.Copy(file, resp.Body); err != nil {
				log.Printf("ERR: Can't copy http body at %v to %v: %v", url, filePath, err)
				continue
			}
			break
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
