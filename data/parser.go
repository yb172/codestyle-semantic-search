package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

type Section struct {
	Document    string `json:"document"`
	ChapterName string `json:"chapter"`
	SectionName string `json:"section"`
	Link        string `json:"link"`
	Contents    string `json:"contents"`
}

func main() {
	url := os.Args[1]
	if len(url) == 0 {
		log.Fatal("Please provide page url as first argument")
	}

	content, err := fetchPage(url)
	if err != nil {
		log.Fatalf("error while fetching page: %s", err)
	}

	sections := parseHTML(url, content)
	var lines []string
	for _, s := range sections {
		jsonData, err := json.Marshal(s)
		if err != nil {
			log.Fatal(err)
		}
		lines = append(lines, string(jsonData))
	}

	os.WriteFile("cpp.jsonl", []byte(strings.Join(lines, "\n")), 0644)
}

func fetchPage(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func parseHTML(docUrl string, htmlContent string) []Section {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal(err)
	}

	var documentName string
	var chapterName string
	var sections []Section
	var currentSection Section

	var parseNode func(*html.Node)
	parseNode = func(n *html.Node) {
		skipParse := false
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1":
				skipParse = true
				documentName = extractText(n)
			case "h2":
				skipParse = true
				chapterName = extractText(n)
				if currentSection.Contents != "" {
					sections = append(sections, currentSection)
				}
				currentSection = Section{
					Document:    documentName,
					ChapterName: chapterName,
					Link:        fmt.Sprintf("%s#%s", docUrl, extractID(n)),
				}
			case "h3":
				{
					skipParse = true
					if currentSection.Contents != "" {
						sections = append(sections, currentSection)
					}
					currentSection = Section{
						Document:    documentName,
						ChapterName: chapterName,
						SectionName: extractText(n),
						Link:        fmt.Sprintf("%s#%s", docUrl, extractID(n)),
					}
				}
			}

		} else if n.Type == html.TextNode && chapterName != "" {
			currentSection.Contents += n.Data
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !skipParse {
				parseNode(c)
			}
		}
	}

	parseNode(doc)

	// Append the last section to the result
	sections = append(sections, currentSection)

	// Strip HTML tags from section contents and clean up newline characters
	for i := range sections {
		sections[i].Contents = cleanUpContents(sections[i].Contents)
	}

	return sections
}

func extractID(n *html.Node) string {
	for _, a := range n.Attr {
		if a.Key == "id" {
			return a.Val
		}
	}
	return ""
}

func cleanUpContents(contents string) string {
	// Trim leading and trailing newline characters
	contents = strings.TrimLeft(contents, "\n")
	contents = strings.TrimRight(contents, "\n")
	contents = strings.ReplaceAll(contents, "\n", " ")
	return contents
}

func extractText(n *html.Node) string {
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			text += c.Data
		}
	}
	return text
}
