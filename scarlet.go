// scarlet.go
package scarlet

import (
   "bufio"
   "bytes"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "strings"
)

func buildAPIRequest(messages []Message, cfg *AppConfig) (*http.Request, error) {
   payload := map[string]any{
      "model":          cfg.Model,
      "messages":       messages,
      "stream":         true,
      "stream_options": map[string]bool{"include_usage": true},
   }

   body, err := json.Marshal(payload)
   if err != nil {
      return nil, fmt.Errorf("marshaling JSON payload: %w", err)
   }

   req, err := http.NewRequest("POST", cfg.APIURL, bytes.NewBuffer(body))
   if err != nil {
      return nil, fmt.Errorf("creating HTTP request: %w", err)
   }

   req.Header.Set("Content-Type", "application/json")
   req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
   req.Header.Set("Accept", "text/event-stream")
   return req, nil
}

func consumeStream(body io.Reader, onToken func(string)) (*Message, error) {
   var fullReasoning, fullContent strings.Builder
   var printedR, reasoningClosed bool

   scanner := bufio.NewScanner(body)

   for scanner.Scan() {
      line := scanner.Text()
      if line == "" || !strings.HasPrefix(line, "data: ") {
         continue
      }

      line = strings.TrimPrefix(line, "data: ")
      if line == "[DONE]" {
         break
      }

      var sr StreamResponse
      if err := json.Unmarshal([]byte(line), &sr); err != nil {
         return nil, fmt.Errorf("error unmarshaling stream chunk: %w\nRaw: %s", err, line)
      }

      for _, choice := range sr.Choices {
         if rc := choice.Delta.ReasoningContent; rc != "" {
            if !printedR {
               if onToken != nil {
                  onToken(`<details class="reasoning" open><summary>Reasoning</summary>`)
               }
               printedR = true
            }
            fullReasoning.WriteString(rc)
            if onToken != nil {
               onToken(escapeHTML(rc))
            }
         }

         if c := choice.Delta.Content; c != "" {
            if printedR && !reasoningClosed {
               if onToken != nil {
                  onToken(`</details>`)
               }
               reasoningClosed = true
            }
            fullContent.WriteString(c)
            if onToken != nil {
               onToken(escapeHTML(c))
            }
         }
      }

      if sr.Usage != nil && sr.Usage.PromptTokens > 0 {
         if printedR && !reasoningClosed {
            if onToken != nil {
               onToken(`</details>`)
            }
            reasoningClosed = true
         }

         stats := fmt.Sprintf(`<div class="token-stats">Input Tokens: %d (%d cached)</div>`,
            sr.Usage.PromptTokens, sr.Usage.PromptTokensDetails.CachedTokens)
         if onToken != nil {
            onToken(stats)
         }
      }
   }

   if printedR && !reasoningClosed {
      if onToken != nil {
         onToken(`</details>`)
      }
   }

   if err := scanner.Err(); err != nil {
      return nil, fmt.Errorf("error reading stream: %w", err)
   }

   return &Message{
      Role:             "assistant",
      Content:          fullContent.String(),
      ReasoningContent: fullReasoning.String(),
   }, nil
}

func processChat(messages []Message, cfg *AppConfig, onToken func(text string)) (*Message, error) {
   req, err := buildAPIRequest(messages, cfg)
   if err != nil {
      return nil, err
   }

   log.Printf("POST %s", cfg.APIURL)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, fmt.Errorf("executing HTTP request: %w", err)
   }
   defer resp.Body.Close()

   if resp.StatusCode != http.StatusOK {
      return nil, fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
   }

   return consumeStream(resp.Body, onToken)
}
