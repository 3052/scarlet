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
   stateOrderedListNestedUL
   stateUnorderedListNestedOL
   stateCodeBlock
   stateOrderedListPending
   stateUnorderedListPending
)

func closeLi(b *strings.Builder) {
   b.WriteString("</li>")
}

func isBlank(line string) bool {
   return strings.TrimSpace(line) == ""
}

func isCodeFence(line string) (int, bool) {
   trimmed := strings.TrimLeft(line, " ")
   indent := len(line) - len(trimmed)
   if strings.TrimSpace(trimmed) == "```" {
      return indent, true
   }
   return 0, false
}

func isIndented(line string) bool {
   return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
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
   content := strings.TrimLeft(rest[1:], " ")
   if len(content) > 0 && content[len(content)-1] == '*' {
      return "", false
   }
   return content, true
}

func openLi(b *strings.Builder, text string) {
   b.WriteString("<li>")
   b.WriteString(renderInline(text))
}

func renderInline(s string) string {
   s = strings.ReplaceAll(s, "-&gt;", "→")

   var result strings.Builder
   inBold := false
   inItalic := false
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
      } else if s[i] == '*' {
         if inItalic {
            result.WriteString("</i>")
         } else {
            result.WriteString("<i>")
         }
         inItalic = !inItalic
         i++
      } else {
         result.WriteByte(s[i])
         i++
      }
   }
   return result.String()
}

func renderMarkdown(s string) string {
   s = html.EscapeString(s)

   lines := strings.Split(s, "\n")
   result := &strings.Builder{}
   state := stateDefault
   var codeLines []string
   codeBlockIndent := 0
   codeBlockReturnState := stateDefault
   liOpen := false

   closeListIfOpen := func() {
      if liOpen {
         closeLi(result)
         liOpen = false
      }
   }

   stripTrailingBr := func() {
      out := result.String()
      if strings.HasSuffix(out, "<br>\n") {
         result.Reset()
         result.WriteString(out[:len(out)-len("<br>\n")])
      }
   }

   for _, line := range lines {
      switch state {
      case stateCodeBlock:
         if _, ok := isCodeFence(line); ok {
            result.WriteString("<pre>")
            result.WriteString(strings.Join(codeLines, "\n"))
            result.WriteString("</pre>\n")
            codeLines = nil
            state = codeBlockReturnState
         } else {
            if len(line) >= codeBlockIndent {
               line = line[codeBlockIndent:]
            }
            codeLines = append(codeLines, line)
         }

      case stateDefault:
         if indent, ok := isCodeFence(line); ok {
            stripTrailingBr()
            codeBlockIndent = indent
            codeBlockReturnState = stateDefault
            state = stateCodeBlock
         } else if strings.TrimSpace(line) == "---" {
            result.WriteString("<hr>\n")
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               result.WriteString("<ol>")
               openLi(result, ordText)
               liOpen = true
               state = stateOrderedList
            } else if isUnord {
               result.WriteString("<ul>")
               openLi(result, unordText)
               liOpen = true
               state = stateUnorderedList
            } else if isBlank(line) {
               result.WriteString("<br>\n")
            } else {
               result.WriteString(renderInline(line))
               result.WriteString("<br>\n")
            }
         }

      case stateOrderedList:
         if indent, ok := isCodeFence(line); ok {
            closeListIfOpen()
            stripTrailingBr()
            codeBlockIndent = indent
            codeBlockReturnState = stateOrderedList
            state = stateCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               closeListIfOpen()
               openLi(result, ordText)
               liOpen = true
            } else if isUnord && isIndented(line) {
               closeListIfOpen()
               result.WriteString("<ul>")
               openLi(result, unordText)
               liOpen = true
               state = stateOrderedListNestedUL
            } else if isUnord {
               closeListIfOpen()
               result.WriteString("</ol><ul>")
               openLi(result, unordText)
               liOpen = true
               state = stateUnorderedList
            } else if isBlank(line) {
               state = stateOrderedListPending
            } else if isIndented(line) {
               result.WriteString("<br>\n")
               result.WriteString(renderInline(strings.TrimLeft(line, " \t")))
            } else {
               closeListIfOpen()
               result.WriteString("</ol>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteByte('\n')
               }
               state = stateDefault
            }
         }

      case stateUnorderedList:
         if indent, ok := isCodeFence(line); ok {
            closeListIfOpen()
            stripTrailingBr()
            codeBlockIndent = indent
            codeBlockReturnState = stateUnorderedList
            state = stateCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isUnord {
               closeListIfOpen()
               openLi(result, unordText)
               liOpen = true
            } else if isOrd && isIndented(line) {
               closeListIfOpen()
               result.WriteString("<ol>")
               openLi(result, ordText)
               liOpen = true
               state = stateUnorderedListNestedOL
            } else if isOrd {
               closeListIfOpen()
               result.WriteString("</ul><ol>")
               openLi(result, ordText)
               liOpen = true
               state = stateOrderedList
            } else if isBlank(line) {
               state = stateUnorderedListPending
            } else if isIndented(line) {
               result.WriteString("<br>\n")
               result.WriteString(renderInline(strings.TrimLeft(line, " \t")))
            } else {
               closeListIfOpen()
               result.WriteString("</ul>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteByte('\n')
               }
               state = stateDefault
            }
         }

      case stateOrderedListNestedUL:
         if indent, ok := isCodeFence(line); ok {
            closeListIfOpen()
            stripTrailingBr()
            codeBlockIndent = indent
            codeBlockReturnState = stateOrderedListNestedUL
            state = stateCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isUnord {
               closeListIfOpen()
               openLi(result, unordText)
               liOpen = true
            } else if isOrd && !isIndented(line) {
               closeListIfOpen()
               result.WriteString("</ul>")
               openLi(result, ordText)
               liOpen = true
               state = stateOrderedList
            } else if isBlank(line) {
               closeListIfOpen()
               result.WriteString("</ul>")
               state = stateOrderedListPending
            } else if isIndented(line) {
               result.WriteString("<br>\n")
               result.WriteString(renderInline(strings.TrimLeft(line, " \t")))
            } else {
               closeListIfOpen()
               result.WriteString("</ul></ol>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteByte('\n')
               }
               state = stateDefault
            }
         }

      case stateUnorderedListNestedOL:
         if indent, ok := isCodeFence(line); ok {
            closeListIfOpen()
            stripTrailingBr()
            codeBlockIndent = indent
            codeBlockReturnState = stateUnorderedListNestedOL
            state = stateCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               closeListIfOpen()
               openLi(result, ordText)
               liOpen = true
            } else if isUnord && !isIndented(line) {
               closeListIfOpen()
               result.WriteString("</ol>")
               openLi(result, unordText)
               liOpen = true
               state = stateUnorderedList
            } else if isBlank(line) {
               closeListIfOpen()
               result.WriteString("</ol>")
               state = stateUnorderedListPending
            } else if isIndented(line) {
               result.WriteString("<br>\n")
               result.WriteString(renderInline(strings.TrimLeft(line, " \t")))
            } else {
               closeListIfOpen()
               result.WriteString("</ol></ul>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteByte('\n')
               }
               state = stateDefault
            }
         }

      case stateOrderedListPending:
         if isBlank(line) {
            closeListIfOpen()
            result.WriteString("</ol>\n")
            state = stateDefault
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               closeListIfOpen()
               openLi(result, ordText)
               liOpen = true
               state = stateOrderedList
            } else if isUnord && isIndented(line) {
               closeListIfOpen()
               result.WriteString("<ul>")
               openLi(result, unordText)
               liOpen = true
               state = stateOrderedListNestedUL
            } else if isUnord {
               closeListIfOpen()
               result.WriteString("</ol><ul>")
               openLi(result, unordText)
               liOpen = true
               state = stateUnorderedList
            } else if isIndented(line) {
               result.WriteString("<br>\n")
               result.WriteString(renderInline(strings.TrimLeft(line, " \t")))
               state = stateOrderedList
            } else if indent, ok := isCodeFence(line); ok {
               closeListIfOpen()
               result.WriteString("</ol>")
               stripTrailingBr()
               codeBlockIndent = indent
               codeBlockReturnState = stateDefault
               state = stateCodeBlock
            } else {
               closeListIfOpen()
               result.WriteString("</ol>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteByte('\n')
               }
               state = stateDefault
            }
         }

      case stateUnorderedListPending:
         if isBlank(line) {
            closeListIfOpen()
            result.WriteString("</ul>\n")
            state = stateDefault
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isUnord {
               closeListIfOpen()
               openLi(result, unordText)
               liOpen = true
               state = stateUnorderedList
            } else if isOrd && isIndented(line) {
               closeListIfOpen()
               result.WriteString("<ol>")
               openLi(result, ordText)
               liOpen = true
               state = stateUnorderedListNestedOL
            } else if isOrd {
               closeListIfOpen()
               result.WriteString("</ul><ol>")
               openLi(result, ordText)
               liOpen = true
               state = stateOrderedList
            } else if isIndented(line) {
               result.WriteString("<br>\n")
               result.WriteString(renderInline(strings.TrimLeft(line, " \t")))
               state = stateUnorderedList
            } else if indent, ok := isCodeFence(line); ok {
               closeListIfOpen()
               result.WriteString("</ul>")
               stripTrailingBr()
               codeBlockIndent = indent
               codeBlockReturnState = stateDefault
               state = stateCodeBlock
            } else {
               closeListIfOpen()
               result.WriteString("</ul>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteByte('\n')
               }
               state = stateDefault
            }
         }
      }
   }

   closeListIfOpen()

   switch state {
   case stateCodeBlock:
      result.WriteString("<pre>")
      result.WriteString(strings.Join(codeLines, "\n"))
      result.WriteString("</pre>\n")
   case stateOrderedList, stateOrderedListPending:
      result.WriteString("</ol>")
   case stateUnorderedList, stateUnorderedListPending:
      result.WriteString("</ul>")
   case stateOrderedListNestedUL:
      result.WriteString("</ul></ol>")
   case stateUnorderedListNestedOL:
      result.WriteString("</ol></ul>")
   }

   stripTrailingBr()

   return result.String()
}
