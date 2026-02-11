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

// Package represents the EPUB OPF package structure (partial, for ISBN and cover)
type Package struct {
	XMLName  xml.Name `xml:"package"`
	Metadata struct {
		Identifiers []struct {
			ID     string `xml:"id,attr"`
			Scheme string `xml:"scheme,attr"`
			Value  string `xml:",chardata"`
		} `xml:"identifier"`
		Meta []struct {
			Name     string `xml:"name,attr"`
			Property string `xml:"property,attr"`
			Refines  string `xml:"refines,attr"`
			Content  string `xml:"content,attr"`
		} `xml:"meta"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID       string `xml:"id,attr"`
			Href     string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
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

	// Find and read container.xml (try common path variants)
	containerFile, err := findAndReadFileFromZip(reader, "META-INF/container.xml")
	if err != nil {
		containerFile, err = findAndReadFileFromZip(reader, "meta-inf/container.xml")
	}
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

	// 1) Prefer identifiers with explicit ISBN scheme
	for _, id := range pkg.Metadata.Identifiers {
		v := strings.TrimSpace(id.Value)
		if v == "" {
			continue
		}
		scheme := strings.ToLower(id.Scheme)
		if scheme == "isbn" || scheme == "isbn-13" || scheme == "isbn-10" {
			if cleaned := sanitizeISBN(v); isValidISBN(cleaned) {
				return cleaned, nil
			}
		}
	}

	// 2) EPUB 3: meta refines="#id" property="identifier-type" -> find identifier with that id
	for _, m := range pkg.Metadata.Meta {
		content := strings.TrimSpace(strings.ToLower(m.Content))
		prop := strings.TrimSpace(strings.ToLower(m.Property))
		if (prop == "identifier-type" || prop == "scheme") && (content == "isbn" || content == "isbn-13" || content == "isbn-10") {
			refinesID := strings.TrimPrefix(strings.TrimSpace(m.Refines), "#")
			for _, id := range pkg.Metadata.Identifiers {
				if id.ID == refinesID {
					v := strings.TrimSpace(id.Value)
					if cleaned := sanitizeISBN(v); isValidISBN(cleaned) {
						return cleaned, nil
					}
					break
				}
			}
		}
	}

	// 3) Fallback: any identifier value that looks like an ISBN (10 or 13 digits)
	for _, id := range pkg.Metadata.Identifiers {
		v := strings.TrimSpace(id.Value)
		cleaned := sanitizeISBN(v)
		if isValidISBN(cleaned) {
			return cleaned, nil
		}
	}

	// 4) Namespace fallback: if no identifiers were unmarshaled, scan raw OPF for identifier/dc:identifier content
	if len(pkg.Metadata.Identifiers) == 0 {
		if isbn := extractISBNFromRawOPF(opfContent); isbn != "" {
			return isbn, nil
		}
	}

	return "", fmt.Errorf("no ISBN found in EPUB metadata")
}

// ExtractCoverFromEPUBBytes extracts the cover image from an EPUB (ZIP). Returns (image bytes, media type, error).
// Looks for <meta name="cover" content="id"/> in OPF metadata and the matching item in manifest.
func ExtractCoverFromEPUBBytes(fileBytes []byte) ([]byte, string, error) {
	if len(fileBytes) == 0 {
		return nil, "", fmt.Errorf("empty file")
	}
	reader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return nil, "", err
	}
	containerFile, err := findAndReadFileFromZip(reader, "META-INF/container.xml")
	if err != nil {
		return nil, "", err
	}
	var container Container
	if err := xml.Unmarshal(containerFile, &container); err != nil {
		return nil, "", err
	}
	if len(container.RootFiles.RootFile) == 0 {
		return nil, "", fmt.Errorf("no rootfile in container")
	}
	opfPath := container.RootFiles.RootFile[0].FullPath
	opfContent, err := findAndReadFileFromZip(reader, opfPath)
	if err != nil {
		return nil, "", err
	}
	var pkg Package
	if err := xml.Unmarshal(opfContent, &pkg); err != nil {
		return nil, "", err
	}
	var coverID string
	for _, m := range pkg.Metadata.Meta {
		if strings.EqualFold(m.Name, "cover") && m.Content != "" {
			coverID = m.Content
			break
		}
	}
	if coverID == "" {
		return nil, "", fmt.Errorf("no cover meta in OPF")
	}
	var coverHref, mediaType string
	for _, item := range pkg.Manifest.Items {
		if item.ID == coverID {
			coverHref = item.Href
			mediaType = item.MediaType
			break
		}
	}
	if coverHref == "" {
		return nil, "", fmt.Errorf("cover id not found in manifest")
	}
	opfDir := opfPath
	if idx := strings.LastIndex(opfPath, "/"); idx >= 0 {
		opfDir = opfPath[:idx+1]
	}
	coverPath := opfDir + coverHref
	coverPath = strings.ReplaceAll(coverPath, "\\", "/")
	coverBytes, err := findAndReadFileFromZip(reader, coverPath)
	if err != nil {
		return nil, "", err
	}
	if mediaType == "" {
		mediaType = "image/jpeg"
	}
	return coverBytes, mediaType, nil
}

// normalizeZipPath replaces backslashes with forward slashes for consistent matching.
func normalizeZipPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

// findAndReadFileFromZip reads a specific file from a zip archive. Path matching is case-insensitive and normalizes backslashes.
func findAndReadFileFromZip(reader *zip.Reader, path string) ([]byte, error) {
	path = normalizeZipPath(path)
	for _, file := range reader.File {
		if normalizeZipPath(file.Name) == path || strings.EqualFold(normalizeZipPath(file.Name), path) {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open zip file entry: %v", err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
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

// isValidISBN returns true if the string (digits only) is a valid ISBN-10 or ISBN-13 length.
func isValidISBN(cleaned string) bool {
	if len(cleaned) == 13 {
		return true // ISBN-13
	}
	if len(cleaned) == 10 {
		return true // ISBN-10
	}
	return false
}

// extractISBNFromRawOPF scans raw OPF XML for identifier-like elements when namespaces prevent normal unmarshaling.
func extractISBNFromRawOPF(opfContent []byte) string {
	s := string(opfContent)
	idents := extractIdentifierContents(s)
	for _, v := range idents {
		v = strings.TrimSpace(v)
		cleaned := sanitizeISBN(v)
		if isValidISBN(cleaned) {
			return cleaned
		}
	}
	return ""
}

// extractIdentifierContents returns text content of all elements whose tag ends with "identifier" (e.g. <identifier>, <dc:identifier>).
func extractIdentifierContents(xmlStr string) []string {
	var out []string
	for i := 0; i < len(xmlStr); i++ {
		if xmlStr[i] != '<' {
			continue
		}
		// Find end of opening tag (allow attributes)
		angle := strings.Index(xmlStr[i:], ">")
		if angle < 0 {
			break
		}
		angle += i
		tagAndAttrs := xmlStr[i+1 : angle]
		// First token is the tag name (e.g. "dc:identifier" or "identifier")
		firstSpace := strings.IndexAny(tagAndAttrs, " \t\n\r")
		tag := tagAndAttrs
		if firstSpace >= 0 {
			tag = tagAndAttrs[:firstSpace]
		}
		if !strings.HasSuffix(tag, "identifier") {
			i = angle
			continue
		}
		contentStart := angle + 1
		// Find closing </...identifier>
		closeIdx := strings.Index(xmlStr[contentStart:], "</")
		if closeIdx < 0 {
			continue
		}
		closeStart := contentStart + closeIdx
		closeEnd := strings.Index(xmlStr[closeStart:], ">")
		if closeEnd < 0 {
			continue
		}
		closeTag := xmlStr[closeStart+2 : closeStart+closeEnd]
		if !strings.HasSuffix(closeTag, "identifier") {
			continue
		}
		content := xmlStr[contentStart:closeStart]
		if !strings.Contains(content, "<") {
			out = append(out, content)
		}
		i = closeStart + closeEnd
	}
	return out
}
