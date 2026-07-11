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
   stateCodeBlock
   stateOrderedListCodeBlock
   stateUnorderedListCodeBlock
)

func isCodeFence(line string) (int, bool) {
   trimmed := strings.TrimLeft(line, " ")
   indent := len(line) - len(trimmed)
   if strings.TrimSpace(trimmed) == "```" {
      return indent, true
   }
   return 0, false
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

func renderInline(s string) string {
   var result strings.Builder
   inBold := false
   inCode := false
   i := 0
   for i < len(s) {
      if s[i] == '`' {
         if inCode {
            result.WriteString("</code>")
         } else {
            result.WriteString("<code>")
         }
         inCode = !inCode
         i++
      } else if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
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

func renderListLi(text string) string {
   return "<li>" + renderInline(text) + "</li>"
}

func renderMarkdown(s string) string {
   s = html.EscapeString(s)

   lines := strings.Split(s, "\n")
   var result strings.Builder
   state := stateDefault
   var codeLines []string
   codeBlockIndent := 0

   for _, line := range lines {
      switch state {
      case stateDefault:
         if indent, ok := isCodeFence(line); ok {
            codeBlockIndent = indent
            state = stateCodeBlock
         } else if strings.TrimSpace(line) == "---" {
            result.WriteString("<hr>\n")
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               result.WriteString("<ol>")
               result.WriteString(renderListLi(ordText))
               state = stateOrderedList
            } else if isUnord {
               result.WriteString("<ul>")
               result.WriteString(renderListLi(unordText))
               state = stateUnorderedList
            } else {
               result.WriteString(renderInline(line))
               result.WriteString("\n")
            }
         }
      case stateOrderedList:
         if indent, ok := isCodeFence(line); ok {
            codeBlockIndent = indent
            state = stateOrderedListCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               result.WriteString(renderListLi(ordText))
            } else if isUnord {
               result.WriteString("</ol>")
               result.WriteString("<ul>")
               result.WriteString(renderListLi(unordText))
               state = stateUnorderedList
            } else {
               result.WriteString("</ol>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteString("\n")
               }
               state = stateDefault
            }
         }
      case stateUnorderedList:
         if indent, ok := isCodeFence(line); ok {
            codeBlockIndent = indent
            state = stateUnorderedListCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isUnord {
               result.WriteString(renderListLi(unordText))
            } else if isOrd {
               result.WriteString("</ul>")
               result.WriteString("<ol>")
               result.WriteString(renderListLi(ordText))
               state = stateOrderedList
            } else {
               result.WriteString("</ul>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteString("\n")
               }
               state = stateDefault
            }
         }
      case stateCodeBlock:
         if _, ok := isCodeFence(line); ok {
            result.WriteString("<pre>")
            result.WriteString(strings.Join(codeLines, "\n"))
            result.WriteString("</pre>\n")
            codeLines = nil
            state = stateDefault
         } else {
            if len(line) >= codeBlockIndent {
               line = line[codeBlockIndent:]
            }
            codeLines = append(codeLines, line)
         }
      case stateOrderedListCodeBlock:
         if _, ok := isCodeFence(line); ok {
            result.WriteString("<pre>")
            result.WriteString(strings.Join(codeLines, "\n"))
            result.WriteString("</pre>\n")
            codeLines = nil
            state = stateOrderedList
         } else {
            if len(line) >= codeBlockIndent {
               line = line[codeBlockIndent:]
            }
            codeLines = append(codeLines, line)
         }
      case stateUnorderedListCodeBlock:
         if _, ok := isCodeFence(line); ok {
            result.WriteString("<pre>")
            result.WriteString(strings.Join(codeLines, "\n"))
            result.WriteString("</pre>\n")
            codeLines = nil
            state = stateUnorderedList
         } else {
            if len(line) >= codeBlockIndent {
               line = line[codeBlockIndent:]
            }
            codeLines = append(codeLines, line)
         }
      }
   }

   switch state {
   case stateCodeBlock, stateOrderedListCodeBlock, stateUnorderedListCodeBlock:
      result.WriteString("<pre>")
      result.WriteString(strings.Join(codeLines, "\n"))
      result.WriteString("</pre>\n")
   case stateOrderedList:
      result.WriteString("</ol>")
   case stateUnorderedList:
      result.WriteString("</ul>")
   }

   return result.String()
}
