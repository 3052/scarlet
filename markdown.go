// markdown.go
package scarlet

import (
   "html"
   "strings"
)

const (
   stateDefault = iota
   stateList
)

func escapeHTML(s string) string {
   return html.EscapeString(s)
}

func isListLine(line string) (string, bool) {
   rest := line
   n := 0
   for len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
      n = n*10 + int(rest[0]-'0')
      rest = rest[1:]
   }
   if n == 0 || len(rest) < 2 || rest[0] != '.' || rest[1] != ' ' {
      return "", false
   }
   return rest[2:], true
}

func renderBold(s string) string {
   var result strings.Builder
   inBold := false
   i := 0
   for i < len(s) {
      if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
         if inBold {
            result.WriteString("</b>")
         } else {
            result.WriteString("<b>")
         }
         inBold = !inBold
         i += 2
      } else {
         result.WriteByte(s[i])
         i++
      }
   }
   return result.String()
}

func renderMarkdown(s string) string {
   s = escapeHTML(s)
   s = strings.ReplaceAll(s, "---", "<hr>")

   lines := strings.Split(s, "\n")
   var result strings.Builder
   state := stateDefault

   for _, line := range lines {
      text, ok := isListLine(line)

      switch state {
      case stateDefault:
         if ok {
            result.WriteString("<ol>")
            result.WriteString("<li>" + renderBold(text) + "</li>")
            state = stateList
         } else {
            result.WriteString(renderBold(line) + "\n")
         }
      case stateList:
         if ok {
            result.WriteString("<li>" + renderBold(text) + "</li>")
         } else {
            result.WriteString("</ol>")
            result.WriteString(renderBold(line) + "\n")
            state = stateDefault
         }
      }
   }

   if state == stateList {
      result.WriteString("</ol>")
   }

   return result.String()
}
