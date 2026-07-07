// markdown.go
package scarlet

import (
   "fmt"
   "html"
   "strings"
)

func isNumberedList(s string) bool {
   idx := strings.Index(s, ". ")
   if idx > 0 && idx <= 3 {
      for i := 0; i < idx; i++ {
         if s[i] < '0' || s[i] > '9' {
            return false
         }
      }
      return true
   }
   return false
}

type Markdown struct {
   inCodeBlock bool
   codeIndent  int
   inList      bool
   prevBlock   bool
   inParagraph bool
}

func (m *Markdown) Render(raw string) string {
   var out strings.Builder
   lines := strings.Split(raw, "\n")

   for _, line := range lines {
      out.WriteString(m.RenderLine(line))
   }

   if m.inList {
      out.WriteString("</ul>")
   }
   if m.inCodeBlock {
      out.WriteString("</pre>")
   }
   if m.inParagraph {
      out.WriteString("</p>")
   }

   return out.String()
}

func (m *Markdown) RenderLine(line string) string {
   trimmed := strings.TrimSpace(line)
   var out strings.Builder

   isListItem := strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") || isNumberedList(trimmed)

   if m.inList && !isListItem && !m.inCodeBlock {
      m.inList = false
      out.WriteString("</ul>")
      m.prevBlock = true
   }

   isBlockStart := isListItem || strings.HasPrefix(trimmed, "```") || trimmed == "<details>" || trimmed == "<details open>" || trimmed == "</details>" || strings.HasPrefix(trimmed, "<summary>") || trimmed == "---" || trimmed == "***" || strings.HasPrefix(trimmed, "#")

   if (isBlockStart || trimmed == "") && m.inParagraph {
      out.WriteString("</p>")
      m.inParagraph = false
   }

   if strings.HasPrefix(trimmed, "```") {
      if !m.inCodeBlock {
         m.inCodeBlock = true
         m.codeIndent = len(line) - len(strings.TrimLeft(line, " "))
         out.WriteString("<pre>")
      } else {
         m.inCodeBlock = false
         m.codeIndent = 0
         out.WriteString("</pre>")
      }
      m.prevBlock = true
      return out.String()
   }

   if m.inCodeBlock {
      trimCount := 0
      for i := 0; i < len(line); i++ {
         if line[i] == ' ' && trimCount < m.codeIndent {
            trimCount++
         } else {
            break
         }
      }
      out.WriteString(html.EscapeString(line[trimCount:]))
      out.WriteString("\n")
      return out.String()
   }

   if trimmed == "<details>" || trimmed == "<details open>" || trimmed == "</details>" {
      out.WriteString(trimmed)
      m.prevBlock = true
      return out.String()
   }

   if strings.HasPrefix(trimmed, "<summary>") && strings.HasSuffix(trimmed, "</summary>") {
      out.WriteString("<summary>")
      out.WriteString(m.parseInline(trimmed[9 : len(trimmed)-10]))
      out.WriteString("</summary>")
      m.prevBlock = true
      return out.String()
   }

   if trimmed == "" {
      return out.String()
   }

   if trimmed == "---" || trimmed == "***" {
      out.WriteString("<hr>")
      m.prevBlock = true
      return out.String()
   }

   if strings.HasPrefix(trimmed, "### ") {
      out.WriteString("<h3>")
      out.WriteString(m.parseInline(strings.TrimPrefix(trimmed, "### ")))
      out.WriteString("</h3>")
      m.prevBlock = true
      return out.String()
   } else if strings.HasPrefix(trimmed, "## ") {
      out.WriteString("<h2>")
      out.WriteString(m.parseInline(strings.TrimPrefix(trimmed, "## ")))
      out.WriteString("</h2>")
      m.prevBlock = true
      return out.String()
   } else if strings.HasPrefix(trimmed, "# ") {
      out.WriteString("<h1>")
      out.WriteString(m.parseInline(strings.TrimPrefix(trimmed, "# ")))
      out.WriteString("</h1>")
      m.prevBlock = true
      return out.String()
   }

   if isListItem {
      if !m.inList {
         m.inList = true
         out.WriteString("<ul>")
      }

      content := trimmed
      if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
         content = trimmed[2:]
      } else {
         idx := strings.Index(trimmed, ". ")
         if idx != -1 {
            content = trimmed[idx+2:]
         }
      }

      leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))
      if leadingSpaces > 0 {
         fmt.Fprintf(&out, "<li style=\"margin-left: %dpx;\">", leadingSpaces*10)
         out.WriteString(m.parseInline(content))
         out.WriteString("</li>")
      } else {
         out.WriteString("<li>")
         out.WriteString(m.parseInline(content))
         out.WriteString("</li>")
      }

      m.prevBlock = true
      return out.String()
   }

   if !m.inParagraph {
      m.inParagraph = true
      out.WriteString("<p>")
   } else {
      out.WriteString("<br>")
   }

   m.prevBlock = false
   out.WriteString(m.parseInline(line))
   return out.String()
}

func (m *Markdown) parseInline(line string) string {
   var out strings.Builder
   inInlineCode := false
   inBold := false
   inItalic := false

   runes := []rune(line)
   for j := 0; j < len(runes); j++ {
      r := runes[j]

      if r == '`' {
         inInlineCode = !inInlineCode
         if inInlineCode {
            out.WriteString("<code>")
         } else {
            out.WriteString("</code>")
         }
         continue
      }

      if !inInlineCode && r == '*' {
         if j < len(runes)-1 && runes[j+1] == '*' {
            inBold = !inBold
            if inBold {
               out.WriteString("<strong>")
            } else {
               out.WriteString("</strong>")
            }
            j++
         } else {
            inItalic = !inItalic
            if inItalic {
               out.WriteString("<em>")
            } else {
               out.WriteString("</em>")
            }
         }
         continue
      }

      if !inInlineCode && r == '-' && j < len(runes)-1 && runes[j+1] == '>' {
         out.WriteString("&rarr;")
         j++
         continue
      }

      switch r {
      case '<':
         out.WriteString("&lt;")
      case '>':
         out.WriteString("&gt;")
      case '&':
         out.WriteString("&amp;")
      case '"':
         out.WriteString("&#34;")
      case '\'':
         out.WriteString("&#39;")
      default:
         out.WriteRune(r)
      }
   }

   if inInlineCode {
      out.WriteString("</code>")
   }
   if inBold {
      out.WriteString("</strong>")
   }
   if inItalic {
      out.WriteString("</em>")
   }

   return out.String()
}
