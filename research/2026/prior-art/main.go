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

   q := url.Values{}
   q.Add("q", `"cached_tokens" "reasoning_content" "prompt_tokens_details" language:go`)
   q.Add("per_page", "100")

   for page := 1; page <= 10; page++ {
      q.Set("page", strconv.Itoa(page))

      req, err := http.NewRequest("GET", baseURL+"?"+q.Encode(), nil)
      if err != nil {
         log.Fatalf("Failed to create request: %v", err)
      }

      req.Header.Set("Authorization", "Bearer "+token)
      req.Header.Set("Accept", "application/vnd.github.v3+json")

      if page > 1 {
         fmt.Printf("\n--- Fetching Search Page %d ---\n", page)
      } else {
         fmt.Println("Querying GitHub Search API...")
      }

      resp, err := doRequest(client, req)
      if err != nil {
         log.Fatalf("Search request failed: %v", err)
      }

      if resp.StatusCode != http.StatusOK {
         body, err := io.ReadAll(resp.Body)
         resp.Body.Close()
         if err != nil {
            log.Fatalf("GitHub API returned status %d (Failed to read error body: %v)", resp.StatusCode, err)
         }
         log.Fatalf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
      }

      var searchResp SearchResponse
      if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
         resp.Body.Close()
         log.Fatalf("Failed to parse search results: %v", err)
      }
      resp.Body.Close()

      if page == 1 {
         fmt.Printf("\n======================================\n")
         fmt.Printf("TOTAL RESULTS FOUND: %d\n", searchResp.TotalCount)
         if searchResp.TotalCount > 1000 {
            fmt.Println("(Note: GitHub API limits search extraction to the first 1000 items)")
         }
         fmt.Printf("======================================\n\n")
         fmt.Printf("--- Fetching Search Page 1 ---\n")
      }

      if len(searchResp.Items) == 0 {
         fmt.Println("No more results found. Finishing up.")
         break
      }

      for _, item := range searchResp.Items {
         repoName := item.Repository.FullName
         repoURL := item.Repository.HTMLURL

         if seenRepos[repoName] {
            continue
         }
         seenRepos[repoName] = true

         // STEP 1: Get the default branch for the repository
         repoReq, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s", repoName), nil)
         if err != nil {
            fmt.Printf("ERROR: Could not create repo info request for %s: %v\n", repoURL, err)
            continue
         }
         repoReq.Header.Set("Authorization", "Bearer "+token)

         repoResp, err := doRequest(client, repoReq)
         if err != nil || repoResp.StatusCode != http.StatusOK {
            fmt.Printf("ERROR: Could not fetch repo info for %s\n", repoURL)
            if repoResp != nil {
               repoResp.Body.Close()
            }
            continue
         }

         var repoInfo RepoInfo
         if err := json.NewDecoder(repoResp.Body).Decode(&repoInfo); err != nil {
            fmt.Printf("ERROR: Failed to decode repo info for %s: %v\n", repoURL, err)
            repoResp.Body.Close()
            continue
         }
         repoResp.Body.Close()

         // STEP 2: Fetch the entire file tree for the repository
         treeReq, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", repoName, repoInfo.DefaultBranch), nil)
         if err != nil {
            fmt.Printf("ERROR: Could not create file tree request for %s: %v\n", repoURL, err)
            continue
         }
         treeReq.Header.Set("Authorization", "Bearer "+token)

         treeResp, err := doRequest(client, treeReq)
         if err != nil || treeResp.StatusCode != http.StatusOK {
            fmt.Printf("ERROR: Could not fetch file tree for %s\n", repoURL)
            if treeResp != nil {
               treeResp.Body.Close()
            }
            continue
         }

         var gitTree GitTree
         if err := json.NewDecoder(treeResp.Body).Decode(&gitTree); err != nil {
            fmt.Printf("ERROR: Failed to decode file tree for %s: %v\n", repoURL, err)
            treeResp.Body.Close()
            continue
         }
         treeResp.Body.Close()

         // STEP 3: Find just the FIRST go.mod file in the tree
         var firstGoMod string
         for _, node := range gitTree.Tree {
            if node.Type == "blob" && strings.HasSuffix(node.Path, "go.mod") && !strings.Contains(node.Path, "vendor/") {
               firstGoMod = node.Path
               break
            }
         }

         if firstGoMod == "" {
            fmt.Printf("FAIL %s (No go.mod found anywhere in repo)\n", repoURL)
            time.Sleep(300 * time.Millisecond)
            continue
         }

         // STEP 4: Download and check that single go.mod file
         safePath := strings.ReplaceAll(firstGoMod, " ", "%20")
         modReq, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repoName, safePath), nil)
         if err != nil {
            fmt.Printf("ERROR: Could not create go.mod request for %s: %v\n", repoURL, err)
            continue
         }
         modReq.Header.Set("Authorization", "Bearer "+token)
         modReq.Header.Set("Accept", "application/vnd.github.v3.raw")

         modResp, err := doRequest(client, modReq)
         if err != nil || modResp.StatusCode != http.StatusOK {
            fmt.Printf("ERROR: Failed to read %s in %s\n", firstGoMod, repoURL)
            if modResp != nil {
               modResp.Body.Close()
            }
            continue
         }

         modContent, err := io.ReadAll(modResp.Body)
         modResp.Body.Close()
         if err != nil {
            fmt.Printf("ERROR: Failed to read go.mod payload for %s: %v\n", repoURL, err)
            continue
         }

         // Final Condition Output
         if strings.Contains(string(modContent), "require") {
            fmt.Printf("FAIL %s\n", repoURL)
         } else {
            fmt.Printf("PASS %s\n", repoURL)
         }

         time.Sleep(300 * time.Millisecond)
      }

      if len(searchResp.Items) < 100 {
         break
      }

      time.Sleep(2 * time.Second)
   }

   fmt.Println("\nScan Complete!")
}
