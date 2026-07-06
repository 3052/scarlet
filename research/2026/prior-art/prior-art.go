package main

import (
   "log"
   "net/http"
   "strconv"
   "time"
)

// doRequest executes the HTTP request and checks strictly for Rate Limit rejections
func doRequest(client *http.Client, req *http.Request) (*http.Response, error) {
   resp, err := client.Do(req)
   if err != nil {
      return nil, err
   }

   // Check for Rate Limit Exceeded (GitHub uses 403 or 429 for this)
   if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
      currentTime := time.Now().Format("15:04:05")
      remaining := resp.Header.Get("X-Ratelimit-Remaining")

      // 1. Primary Rate Limit (e.g., Search limit of 30/min exhausted)
      if remaining == "0" {
         resetStr := resp.Header.Get("X-Ratelimit-Reset")
         resetUnix, err := strconv.ParseInt(resetStr, 10, 64)
         var resetTime string
         if err == nil {
            resetTime = time.Unix(resetUnix, 0).Format("15:04:05")
         } else {
            resetTime = "unknown time"
         }
         log.Fatalf("\n[!] RATE LIMIT REJECT at %s: Primary limit exhausted. Limit resets at %s.\n", currentTime, resetTime)
      }

      // 2. Secondary/Abuse Rate Limit (Triggered if making requests too rapidly)
      retryAfter := resp.Header.Get("Retry-After")
      if retryAfter != "" {
         log.Fatalf("\n[!] RATE LIMIT REJECT at %s: Secondary limit triggered. Server says to wait %s seconds.\n", currentTime, retryAfter)
      }

      // 3. Fallback for limits without standard headers
      log.Fatalf("\n[!] RATE LIMIT REJECT at %s: Server rejected the request due to rate limiting.\n", currentTime)
   }

   return resp, nil
}

type GitTree struct {
   Tree []struct {
      Path string `json:"path"`
      Type string `json:"type"`
   } `json:"tree"`
}

type RepoInfo struct {
   DefaultBranch   string `json:"default_branch"`
   StargazersCount int    `json:"stargazers_count"` // Added to check for zero-star repos
}

type SearchResponse struct {
   TotalCount int `json:"total_count"`
   Items      []struct {
      Repository struct {
         FullName string `json:"full_name"`
         HTMLURL  string `json:"html_url"`
      } `json:"repository"`
   } `json:"items"`
}
