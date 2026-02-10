package utils

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Container represents the EPUB container.xml structure
type Container struct {
	XMLName   xml.Name `xml:"container"`
	RootFiles struct {
		RootFile []struct {
			FullPath  string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

// Package represents the EPUB OPF package structure
type Package struct {
	XMLName  xml.Name `xml:"package"`
	Metadata struct {
		Identifiers []struct {
			Scheme string `xml:"scheme,attr"`
			Value  string `xml:",chardata"`
		} `xml:"identifier"`
	} `xml:"metadata"`
}

// GoogleBooksResponse represents the response structure from Google Books API
type GoogleBooksResponse struct {
	Items []struct {
		VolumeInfo struct {
			Title       string   `json:"title"`
			Authors     []string `json:"authors"`
			Publisher   string   `json:"publisher"`
			PageCount   int      `json:"pageCount"`
			Categories  []string `json:"categories"`
			ISBN13      string   `json:"industryIdentifiers"`
			PreviewLink string   `json:"previewLink"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

// ExtractISBNFromMultipartFile processes an uploaded file and returns its ISBN
func ExtractISBNFromMultipartFile(file io.Reader) (string, error) {
	if file == nil {
		return "", fmt.Errorf("received nil file")
	}

	// Read the file into a buffer first to check if it's empty
	buffer := new(bytes.Buffer)
	size, err := io.Copy(buffer, file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	if size == 0 {
		return "", fmt.Errorf("uploaded file is empty")
	}

	fileBytes := buffer.Bytes()
	reader, err := zip.NewReader(bytes.NewReader(fileBytes), size)
	if err != nil {
		return "", fmt.Errorf("invalid EPUB file (not a valid ZIP): %v", err)
	}

	// Find and read container.xml
	containerFile, err := findAndReadFileFromZip(reader, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("failed to read container.xml: %v", err)
	}

	var container Container
	if err := xml.Unmarshal(containerFile, &container); err != nil {
		return "", fmt.Errorf("failed to parse container.xml: %v", err)
	}

	if len(container.RootFiles.RootFile) == 0 {
		return "", fmt.Errorf("no rootfile found in container.xml")
	}

	// Read the OPF file
	opfPath := container.RootFiles.RootFile[0].FullPath
	opfContent, err := findAndReadFileFromZip(reader, opfPath)
	if err != nil {
		return "", fmt.Errorf("failed to read OPF file: %v", err)
	}

	var pkg Package
	if err := xml.Unmarshal(opfContent, &pkg); err != nil {
		return "", fmt.Errorf("failed to parse OPF file: %v", err)
	}

	// Look for ISBN in identifiers
	for _, identifier := range pkg.Metadata.Identifiers {
		scheme := strings.ToLower(identifier.Scheme)
		if scheme == "isbn" || scheme == "isbn-13" || scheme == "isbn-10" {
			return sanitizeISBN(identifier.Value), nil
		}
	}

	return "", fmt.Errorf("no ISBN found in EPUB metadata")
}


// findAndReadFileFromZip reads a specific file from a zip archive
func findAndReadFileFromZip(reader *zip.Reader, path string) ([]byte, error) {
	for _, file := range reader.File {
		if file.Name == path {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open zip file entry: %v", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to read zip file entry: %v", err)
			}
			return content, nil
		}
	}
	return nil, fmt.Errorf("file not found in zip: %s", path)
}

// sanitizeISBN removes any non-digit characters from the ISBN
func sanitizeISBN(isbn string) string {
	var cleaned strings.Builder
	for _, r := range isbn {
		if r >= '0' && r <= '9' {
			cleaned.WriteRune(r)
		}
	}
	return cleaned.String()
}
