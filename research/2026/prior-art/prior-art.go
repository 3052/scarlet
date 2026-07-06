package main

import (
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "net/url"
   "os"
   "strconv"
   "strings"
   "time"
)

func main() {
   token := os.Getenv("GITHUB_TOKEN")
   if token == "" {
      log.Fatal("Error: Please set the GITHUB_TOKEN environment variable.")
   }

   client := &http.Client{Timeout: 10 * time.Second}
   seenRepos := make(map[string]bool)

   baseURL := "https://api.github.com/search/code"

   // Set up our base query parameters
   q := url.Values{}
   q.Add("q", `"cached_tokens" language:go`)
   q.Add("per_page", "100")

   // GitHub limits search results to 1000 items total (10 pages of 100)
   for page := 1; page <= 10; page++ {
      q.Set("page", strconv.Itoa(page))

      req, err := http.NewRequest("GET", baseURL+"?"+q.Encode(), nil)
      if err != nil {
         log.Fatalf("Failed to create request: %v", err)
      }

      req.Header.Set("Authorization", "Bearer "+token)
      req.Header.Set("Accept", "application/vnd.github.v3+json")

      fmt.Printf("\n--- Fetching Search Page %d ---\n", page)

      resp, err := client.Do(req)
      if err != nil {
         log.Fatalf("Search request failed: %v", err)
      }

      if resp.StatusCode != http.StatusOK {
         body, _ := io.ReadAll(resp.Body)
         resp.Body.Close()
         log.Fatalf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
      }

      var searchResp SearchResponse
      if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
         resp.Body.Close()
         log.Fatalf("Failed to parse search results: %v", err)
      }
      resp.Body.Close()

      if len(searchResp.Items) == 0 {
         fmt.Println("No more results found. Finishing up.")
         break
      }

      fmt.Printf("Processing %d items from page %d...\n", len(searchResp.Items), page)

      // Iterate through the current page's matches
      for _, item := range searchResp.Items {
         repoName := item.Repository.FullName
         repoURL := item.Repository.HTMLURL

         // Deduplicate: If we've already checked this repo, skip it
         if seenRepos[repoName] {
            continue
         }
         seenRepos[repoName] = true

         // Fetch the raw go.mod file
         goModURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/go.mod", repoName)
         modReq, err := http.NewRequest("GET", goModURL, nil)
         if err != nil {
            fmt.Printf("ERROR: Could not create request for %s\n", repoURL)
            continue
         }

         modReq.Header.Set("Authorization", "Bearer "+token)
         modReq.Header.Set("Accept", "application/vnd.github.v3.raw")

         modResp, err := client.Do(modReq)
         if err != nil {
            fmt.Printf("ERROR: Request failed for %s\n", repoURL)
            continue
         }

         if modResp.StatusCode == http.StatusNotFound {
            fmt.Printf("PASS %s (No go.mod found)\n", repoURL)
            modResp.Body.Close()
            continue
         } else if modResp.StatusCode != http.StatusOK {
            fmt.Printf("ERROR: Failed to fetch go.mod for %s (Status: %d)\n", repoURL, modResp.StatusCode)
            modResp.Body.Close()
            continue
         }

         modContent, err := io.ReadAll(modResp.Body)
         modResp.Body.Close()
         if err != nil {
            fmt.Printf("ERROR: Failed to read go.mod for %s\n", repoURL)
            continue
         }

         // Check for "require"
         if strings.Contains(string(modContent), "require") {
            fmt.Printf("FAIL %s\n", repoURL)
         } else {
            fmt.Printf("PASS %s\n", repoURL)
         }

         // Sleep to respect GitHub's secondary rate limits
         time.Sleep(500 * time.Millisecond)
      }

      // If the API returned fewer than 100 items, we've hit the last page
      if len(searchResp.Items) < 100 {
         break
      }

      // Optional: wait a couple of seconds between page searches to avoid hitting the 10 requests/minute search rate limit
      time.Sleep(2 * time.Second)
   }

   fmt.Println("\nScan Complete!")
}

type SearchResponse struct {
   Items []struct {
      Repository struct {
         FullName string `json:"full_name"`
         HTMLURL  string `json:"html_url"`
      } `json:"repository"`
   } `json:"items"`
}
