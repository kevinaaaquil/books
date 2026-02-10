package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Open Library ISBN response (subset we use)
type openLibraryISBNResp struct {
	Title          string   `json:"title"`
	Publishers     []string `json:"publishers"`
	PublishDate    string   `json:"publish_date"`
	NumberOfPages  int      `json:"number_of_pages"`
	EditionName    string   `json:"edition_name"`
	ISBN10         []string `json:"isbn_10"`
	ISBN13         []string `json:"isbn_13"`
	Covers         []int    `json:"covers"`
	Works          []struct {
		Key string `json:"key"`
	} `json:"works"`
}

// BookMetadata is the normalized metadata we store and return.
type BookMetadata struct {
	Title       string
	Authors     []string
	Publisher   string
	PublishDate string
	ISBN        string
	PageCount   int
	CoverURL    string
	Edition     string
}

const openLibraryBase = "https://openlibrary.org"

// FetchMetadataByISBN fetches book metadata from Open Library API by ISBN.
func FetchMetadataByISBN(isbn string) (*BookMetadata, error) {
	isbn = strings.TrimSpace(isbn)
	if isbn == "" {
		return nil, fmt.Errorf("isbn is required")
	}
	u := openLibraryBase + "/isbn/" + url.PathEscape(isbn) + ".json"
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open library returned %d", resp.StatusCode)
	}
	var ol openLibraryISBNResp
	if err := json.NewDecoder(resp.Body).Decode(&ol); err != nil {
		return nil, err
	}
	meta := &BookMetadata{
		Title:       ol.Title,
		PublishDate: ol.PublishDate,
		PageCount:   ol.NumberOfPages,
		Edition:     ol.EditionName,
		ISBN:        isbn,
	}
	if len(ol.Publishers) > 0 {
		meta.Publisher = ol.Publishers[0]
	}
	if len(ol.ISBN13) > 0 {
		meta.ISBN = ol.ISBN13[0]
	} else if len(ol.ISBN10) > 0 {
		meta.ISBN = ol.ISBN10[0]
	}
	if len(ol.Covers) > 0 {
		meta.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", ol.Covers[0])
	}
	// Optionally fetch authors from work; for now leave empty if not in edition response
	if len(ol.Works) > 0 {
		authors, _ := fetchAuthorsForWork(ol.Works[0].Key)
		meta.Authors = authors
	}
	return meta, nil
}

type workResp struct {
	Authors []struct {
		Author struct {
			Key string `json:"key"`
		} `json:"author"`
	} `json:"authors"`
}

type authorResp struct {
	Name string `json:"name"`
}

func fetchAuthorsForWork(workKey string) ([]string, error) {
	u := openLibraryBase + workKey + ".json"
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var work workResp
	if err := json.NewDecoder(resp.Body).Decode(&work); err != nil {
		return nil, err
	}
	var names []string
	for _, a := range work.Authors {
		authorURL := openLibraryBase + a.Author.Key + ".json"
		r, err := http.Get(authorURL)
		if err != nil {
			continue
		}
		var author authorResp
		_ = json.NewDecoder(r.Body).Decode(&author)
		r.Body.Close()
		if author.Name != "" {
			names = append(names, author.Name)
		}
	}
	return names, nil
}
