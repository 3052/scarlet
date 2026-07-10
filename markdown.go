// markdown.go
package scarlet

import (
   "html"
   "strings"
)

const (
   stateDefault = iota
   stateOrderedList
   stateUnorderedList
)

func escapeHTML(s string) string {
   return html.EscapeString(s)
}

func isOrderedListLine(line string) (string, bool) {
   rest := strings.TrimLeft(line, " \t")
   n := 0
   for len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9' {
      n = n*10 + int(rest[0]-'0')
      rest = rest[1:]
   }
   if n == 0 || len(rest) < 2 || rest[0] != '.' {
      return "", false
   }
   rest = strings.TrimLeft(rest[1:], " ")
   return rest, true
}

func isUnorderedListLine(line string) (string, bool) {
   rest := strings.TrimLeft(line, " \t")
   if len(rest) == 0 || rest[0] != '*' {
      return "", false
   }
   rest = strings.TrimLeft(rest[1:], " ")
   return rest, true
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
      ordText, isOrd := isOrderedListLine(line)
      unordText, isUnord := isUnorderedListLine(line)

      switch state {
      case stateDefault:
         if isOrd {
            result.WriteString("<ol>")
            result.WriteString("<li>" + renderBold(ordText) + "</li>")
            state = stateOrderedList
         } else if isUnord {
            result.WriteString("<ul>")
            result.WriteString("<li>" + renderBold(unordText) + "</li>")
            state = stateUnorderedList
         } else {
            result.WriteString(renderBold(line) + "\n")
         }
      case stateOrderedList:
         if isOrd {
            result.WriteString("<li>" + renderBold(ordText) + "</li>")
         } else if isUnord {
            result.WriteString("</ol>")
            result.WriteString("<ul>")
            result.WriteString("<li>" + renderBold(unordText) + "</li>")
            state = stateUnorderedList
         } else {
            result.WriteString("</ol>")
            result.WriteString(renderBold(line) + "\n")
            state = stateDefault
         }
      case stateUnorderedList:
         if isUnord {
            result.WriteString("<li>" + renderBold(unordText) + "</li>")
         } else if isOrd {
            result.WriteString("</ul>")
            result.WriteString("<ol>")
            result.WriteString("<li>" + renderBold(ordText) + "</li>")
            state = stateOrderedList
         } else {
            result.WriteString("</ul>")
            result.WriteString(renderBold(line) + "\n")
            state = stateDefault
         }
      }
   }

   switch state {
   case stateOrderedList:
      result.WriteString("</ol>")
   case stateUnorderedList:
      result.WriteString("</ul>")
   }

   return result.String()
}
