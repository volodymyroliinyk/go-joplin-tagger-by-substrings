// Run: `go run main.go --tag_name="links" --contains_substring="https://"
package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

// JOPLIN API CONFIGURATION
const (
    JOPLIN_API_BASE = "http://localhost:41184"
    // REPLACE THIS WITH YOUR TOKEN
    JOPLIN_TOKEN = "YOUR_JOPLIN_API_TOKEN"
)

// Structures for parsing API responses
type Tag struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type Note struct {
    ID    string `json:"id"`
    Title string `json:"title"`
    Body  string `json:"body"`
}

type CollectionResponse struct {
    Items []json.RawMessage `json:"items"`
}

// Custom structure for handling multiple flag values
type ArrayFlags []string

func (i *ArrayFlags) String() string {
    return strings.Join(*i, ", ")
}

func (i *ArrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}

// --- API FUNCTIONS (Mostly unchanged from main.go) ---

// fetchData makes a GET request to the Joplin API
func fetchData(endpoint string) ([]byte, error) {
    separator := "?"
    if strings.Contains(endpoint, "?") {
        separator = "&"
    }

    url := fmt.Sprintf("%s%s%stoken=%s", JOPLIN_API_BASE, endpoint, separator, JOPLIN_TOKEN)

    client := http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("error executing request to %s: %w. Check that Joplin and the API are working", endpoint, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API error: status %d for %s", resp.StatusCode, endpoint)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("error reading response body: %w", err)
    }
    return body, nil
}

// getAllNotes gets a list of all notes using pagination (Note: now requests 'body' by default)
func getAllNotes() ([]Note, error) {
    fmt.Println("-> Get all notes (with pagination)...")
    var notes []Note
    page := 1

    for {
        endpoint := fmt.Sprintf("/notes?page=%d&fields=id,title,body", page)
        body, err := fetchData(endpoint)
        if err != nil {
            return nil, err
        }

        var response CollectionResponse
        if err := json.Unmarshal(body, &response); err != nil {
            return nil, fmt.Errorf("error parsing the notes on the page %d: %w", page, err)
        }

        if len(response.Items) == 0 {
            break
        }

        for _, rawItem := range response.Items {
            var note Note
            if err := json.Unmarshal(rawItem, &note); err != nil {
                log.Printf("Error parsing the note: %v", err)
                continue
            }
            notes = append(notes, note)
        }

        page++
    }

    fmt.Printf("   Found %d notes.\n", len(notes))
    return notes, nil
}

// getAllTags gets a list of all tags using pagination
func getAllTags() ([]Tag, error) {
    fmt.Println("-> Getting all tags (with pagination)...")
    var tags []Tag
    page := 1

    for {
        endpoint := fmt.Sprintf("/tags?page=%d", page)
        body, err := fetchData(endpoint)
        if err != nil {
            return nil, err
        }

        var response CollectionResponse
        if err := json.Unmarshal(body, &response); err != nil {
            return nil, fmt.Errorf("tag parsing error on page %d: %w", page, err)
        }

        if len(response.Items) == 0 {
            break
        }

        for _, rawItem := range response.Items {
            var tag Tag
            if err := json.Unmarshal(rawItem, &tag); err != nil {
                log.Printf("Error parsing tag: %v", err)
                continue
            }
            tags = append(tags, tag)
        }

        page++
    }

    fmt.Printf("   Found %d tags.\n", len(tags))
    return tags, nil
}

// getNoteTags gets the IDs of the tags already attached to the note
func getNoteTags(noteID string) (map[string]bool, error) {
    endpoint := fmt.Sprintf("/notes/%s/tags", noteID)
    body, err := fetchData(endpoint)
    if err != nil {
        return nil, err
    }

    var response CollectionResponse
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, fmt.Errorf("error parsing note tags: %w", err)
    }

    existingTags := make(map[string]bool)
    for _, rawItem := range response.Items {
        var tag Tag
        if err := json.Unmarshal(rawItem, &tag); err != nil {
            log.Printf("Error parsing an existing tag: %v", err)
            continue
        }
        existingTags[tag.ID] = true
    }
    return existingTags, nil
}

// associateTag attaches the tag to the note
func associateTag(noteID, tagID string) error {
    url := fmt.Sprintf("%s/tags/%s/notes?token=%s", JOPLIN_API_BASE, tagID, JOPLIN_TOKEN)

    payload := map[string]string{"id": noteID}
    jsonPayload, _ := json.Marshal(payload)

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
    if err != nil {
        return fmt.Errorf("error creating POST request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("POST request execution error: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        if resp.StatusCode == http.StatusConflict {
            return nil // Already attached
        }
        return fmt.Errorf("error attaching tag: status %d", resp.StatusCode)
    }

    return nil
}

// createTag creates a new tag and returns its ID
func createTag(title string) (string, error) {
    url := fmt.Sprintf("%s/tags?token=%s", JOPLIN_API_BASE, JOPLIN_TOKEN)

    payload := map[string]string{"title": title}
    jsonPayload, _ := json.Marshal(payload)

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
    if err != nil {
        return "", fmt.Errorf("error creating POST request for tag creation: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("POST request execution error for tag creation: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("error creating tag: status %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("error reading response body after tag creation: %w", err)
    }

    var newTag Tag
    if err := json.Unmarshal(body, &newTag); err != nil {
        return "", fmt.Errorf("error parsing new tag response: %w", err)
    }

    return newTag.ID, nil
}

// --- MAIN LOGIC ---

func main() {
    if JOPLIN_TOKEN == "YOUR_JOPLIN_API_TOKEN" {
        log.Fatal("ERROR: Please replace stub 'YOUR_JOPLIN_API_TOKEN' with your actual API token.")
    }

    var tagName string
    var containsSubstrings ArrayFlags

    // 1. Parsing command line arguments
    flag.StringVar(&tagName, "tag_name", "", "The name of the tag to create or find.")
    flag.Var(&containsSubstrings, "contains_substring", "A substring that the note body must contain (can be used multiple times).")
    flag.Parse()

    if tagName == "" || len(containsSubstrings) == 0 {
        fmt.Println("Usage:")
        fmt.Println("  go run tagger.go --tag_name=\"[Tag Name]\" --contains_substring=\"[Substring 1]\" --contains_substring=\"[Substring 2]\" ...")
        fmt.Println("\nExample:")
        fmt.Println("  go run tagger.go --tag_name=\"Important Project\" --contains_substring=\"Deadline\" --contains_substring=\"Project X\"")
        os.Exit(1)
    }

    fmt.Printf("Tag Name: **%s**\n", tagName)
    fmt.Printf("Required Substrings (AND logic): **%s**\n", strings.Join(containsSubstrings, " AND "))
    fmt.Println("---")

    // 2. Find or Create Tag
    allTags, err := getAllTags()
    if err != nil {
        log.Fatalf("Critical error getting tags: %v", err)
    }

    var targetTagID string
    // Search for an existing tag
    for _, tag := range allTags {
        if tag.Title == tagName {
            targetTagID = tag.ID
            fmt.Printf("-> Found existing tag: '%s' (ID: %s)\n", tagName, targetTagID)
            break
        }
    }

    // Create a new tag if not found
    if targetTagID == "" {
        fmt.Printf("-> Tag '%s' not found. Creating a new one...\n", tagName)
        targetTagID, err = createTag(tagName)
        if err != nil {
            log.Fatalf("Critical error creating tag '%s': %v", tagName, err)
        }
        fmt.Printf("   [SUCCESS] New tag created with ID: %s\n", targetTagID)
    }

    // 3. Get all notes
    allNotes, err := getAllNotes()
    if err != nil {
        log.Fatalf("Critical error getting notes: %v", err)
    }

    // 4. Scan notes and apply tag
    fmt.Println("\n-> Start scanning and tagging notes...")
    totalTagsAdded := 0

    for i, note := range allNotes {
        // Preparation of the note body for search (conversion to lower case for case-insensitive search)
        noteBodyLower := strings.ToLower(note.Body)

        // Check 1: Do all substrings exist in the note body? (AND logic)
        matchesAllSubstrings := true
        for _, sub := range containsSubstrings {
            // Search for the substring in the body (case-insensitive)
            if !strings.Contains(noteBodyLower, strings.ToLower(sub)) {
                matchesAllSubstrings = false
                break // Failed the AND condition
            }
        }

        if matchesAllSubstrings {
            // log.Printf("--- Note processing %d/%d: '%s' (ID: %s) ---", i+1, len(allNotes), note.Title, note.ID)

            // Check 2: Is the tag already attached?
            existingTagIDs, err := getNoteTags(note.ID)
            if err != nil {
                log.Printf("[Skip note %d/%d '%s']: Failed to get existing tags: %v", i+1, len(allNotes), note.Title, err)
                continue
            }

            if existingTagIDs[targetTagID] {
                // fmt.Printf("   [SKIP] Note '%s' already has tag '%s'.\n", note.Title, tagName)
                continue // Tag is already attached, nothing to do
            }

            // Check 3: Apply the tag
            err = associateTag(note.ID, targetTagID)
            if err != nil {
                log.Printf("[ERROR] Failed to attach tag '%s' to note'%s': %v", tagName, note.Title, err)
            } else {
                fmt.Printf("   [SUCCESS] Tagged note **'%s'** with **'%s'**.\n", note.Title, tagName)
                totalTagsAdded++
            }
        }
    }

    fmt.Printf("\nDONE! Added a total of **%d** new tags to notes.\n", totalTagsAdded)
}
