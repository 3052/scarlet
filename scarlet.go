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

func flushBuffers(buf *string, md *Markdown, onToken func(string)) {
   for {
      idx := strings.IndexByte(*buf, '\n')
      if idx == -1 {
         break
      }
      lineChunk := (*buf)[:idx]
      *buf = (*buf)[idx+1:]
      if onToken != nil {
         onToken(md.RenderLine(lineChunk))
      }
   }
}

func flushRemaining(buf *string, md *Markdown, onToken func(string)) {
   if *buf != "" {
      if onToken != nil {
         onToken(md.RenderLine(*buf))
      }
      *buf = ""
   }
   if md.inList {
      if onToken != nil {
         onToken("</ul>")
      }
      md.inList = false
   }
   if md.inCodeBlock {
      if onToken != nil {
         onToken("</pre>")
      }
      md.inCodeBlock = false
   }
   if md.inParagraph {
      if onToken != nil {
         onToken("</p>")
      }
      md.inParagraph = false
   }
}

func consumeStream(body io.Reader, onToken func(string)) (*Message, error) {
   var fullReasoning, fullContent strings.Builder
   var rBuf, cBuf string
   var printedR, printedC bool

   rMd, cMd := &Markdown{}, &Markdown{}
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
            rBuf += rc
            flushBuffers(&rBuf, rMd, onToken)
         }

         if c := choice.Delta.Content; c != "" {
            if printedR && !printedC {
               if onToken != nil {
                  onToken(`</details>`)
               }
               printedC = true
            }
            fullContent.WriteString(c)
            cBuf += c
            flushBuffers(&cBuf, cMd, onToken)
         }
      }

      if sr.Usage != nil && sr.Usage.PromptTokens > 0 {
         if printedR && !printedC {
            flushRemaining(&rBuf, rMd, onToken)
            if onToken != nil {
               onToken(`</details>`)
            }
            printedC = true
         }

         flushRemaining(&cBuf, cMd, onToken)

         stats := fmt.Sprintf(`<div class="token-stats">Input Tokens: %d (%d cached)</div>`,
            sr.Usage.PromptTokens, sr.Usage.PromptTokensDetails.CachedTokens)
         if onToken != nil {
            onToken(stats)
         }
      }
   }

   if printedR && !printedC {
      if onToken != nil {
         onToken(`</details>`)
      }
   }
   flushRemaining(&cBuf, cMd, onToken)

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
