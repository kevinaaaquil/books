package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const googleBooksBase = "https://www.googleapis.com/books/v1/volumes"

// googleBooksClient has a short timeout so slow/hung responses don't block uploads.
var googleBooksClient = &http.Client{Timeout: 15 * time.Second}

// googleBooksVolumesResp is the response from GET /volumes?q=isbn:...
type googleBooksVolumesResp struct {
	TotalItems int `json:"totalItems"`
	Items      []struct {
		VolumeInfo struct {
			Title         string   `json:"title"`
			Subtitle      string   `json:"subtitle"`
			Authors       []string `json:"authors"`
			Publisher     string   `json:"publisher"`
			PublishedDate string   `json:"publishedDate"`
			Description   string   `json:"description"`
			PageCount     int      `json:"pageCount"`
			Categories    []string `json:"categories"`
			ImageLinks    struct {
				SmallThumbnail string `json:"smallThumbnail"`
				Thumbnail      string `json:"thumbnail"`
			} `json:"imageLinks"`
			IndustryIdentifiers []struct {
				Type       string `json:"type"`
				Identifier string `json:"identifier"`
			} `json:"industryIdentifiers"`
			AverageRating float64 `json:"averageRating"`
			RatingsCount  int     `json:"ratingsCount"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

// BookMetadata is the normalized metadata we store and return.
type BookMetadata struct {
	Title         string
	Authors       []string
	Publisher     string
	PublishDate   string
	ISBN          string
	PageCount     int
	CoverURL      string
	ThumbnailURL  string
	Edition       string
	Preface       string   // description
	Category      string
	Categories    []string
	RatingAverage float64
	RatingCount   int
}

// FetchMetadataByISBN fetches book metadata from Google Books API by ISBN.
func FetchMetadataByISBN(isbn string) (*BookMetadata, error) {
	isbn = strings.ReplaceAll(strings.TrimSpace(isbn), "-", "")
	if isbn == "" {
		return nil, fmt.Errorf("isbn is required")
	}
	q := url.Values{}
	q.Set("q", "isbn:"+isbn)
	u := googleBooksBase + "?" + q.Encode()
	resp, err := googleBooksClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google books returned %d", resp.StatusCode)
	}
	var data googleBooksVolumesResp
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.TotalItems == 0 || len(data.Items) == 0 {
		return nil, fmt.Errorf("no volume found for isbn %s", isbn)
	}
	vi := data.Items[0].VolumeInfo
	meta := &BookMetadata{
		Title:         vi.Title,
		Authors:       vi.Authors,
		Publisher:     vi.Publisher,
		PublishDate:   vi.PublishedDate,
		PageCount:     vi.PageCount,
		Categories:    vi.Categories,
		RatingAverage: vi.AverageRating,
		RatingCount:   vi.RatingsCount,
		ISBN:          isbn,
	}
	if vi.Subtitle != "" {
		meta.Title = meta.Title + ": " + vi.Subtitle
	}
	if len(vi.IndustryIdentifiers) > 0 {
		for _, id := range vi.IndustryIdentifiers {
			if id.Type == "ISBN_13" || id.Type == "ISBN_10" {
				meta.ISBN = id.Identifier
				break
			}
		}
	}
	if len(vi.Categories) > 0 {
		meta.Category = vi.Categories[0]
	}
	// Use Open Library covers by ISBN (no captcha); Google Books image URLs often require captcha
	if meta.ISBN != "" {
		meta.CoverURL = openLibraryCoverURL(meta.ISBN, "L")
		meta.ThumbnailURL = openLibraryCoverURL(meta.ISBN, "M")
	}
	meta.Preface = strings.TrimSpace(vi.Description)
	return meta, nil
}

// openLibraryCoverURL returns a direct cover image URL by ISBN. Size: S (small), M (medium), L (large). No captcha.
func openLibraryCoverURL(isbn, size string) string {
	isbn = strings.TrimSpace(isbn)
	if isbn == "" {
		return ""
	}
	// Strip hyphens for a clean URL; Open Library accepts both
	clean := strings.ReplaceAll(isbn, "-", "")
	if clean == "" {
		return ""
	}
	return "https://covers.openlibrary.org/b/isbn/" + url.PathEscape(clean) + "-" + size + ".jpg"
}
