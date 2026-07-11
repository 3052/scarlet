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
   rest = strings.TrimLeft(rest[1:], " ")
   return rest, true
}

func renderInline(s string) string {
   s = strings.ReplaceAll(s, "-&gt;", "→")

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
   codeBlockReturnState := stateDefault

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
            codeBlockReturnState = stateOrderedList
            state = stateCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               result.WriteString(renderListLi(ordText))
            } else if isUnord && isIndented(line) {
               result.WriteString("<ul>")
               result.WriteString(renderListLi(unordText))
               state = stateOrderedListNestedUL
            } else if isUnord {
               result.WriteString("</ol>")
               result.WriteString("<ul>")
               result.WriteString(renderListLi(unordText))
               state = stateUnorderedList
            } else if isBlank(line) {
               state = stateOrderedListPending
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
            codeBlockReturnState = stateUnorderedList
            state = stateCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isUnord {
               result.WriteString(renderListLi(unordText))
            } else if isOrd && isIndented(line) {
               result.WriteString("<ol>")
               result.WriteString(renderListLi(ordText))
               state = stateUnorderedListNestedOL
            } else if isOrd {
               result.WriteString("</ul>")
               result.WriteString("<ol>")
               result.WriteString(renderListLi(ordText))
               state = stateOrderedList
            } else if isBlank(line) {
               state = stateUnorderedListPending
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

      case stateOrderedListNestedUL:
         if indent, ok := isCodeFence(line); ok {
            codeBlockIndent = indent
            codeBlockReturnState = stateOrderedListNestedUL
            state = stateCodeBlock
         } else {
            unordText, isUnord := isUnorderedListLine(line)
            ordText, isOrd := isOrderedListLine(line)
            if isUnord {
               result.WriteString(renderListLi(unordText))
            } else if isOrd && !isIndented(line) {
               result.WriteString("</ul>")
               result.WriteString(renderListLi(ordText))
               state = stateOrderedList
            } else if isBlank(line) {
               result.WriteString("</ul>")
               state = stateOrderedListPending
            } else {
               result.WriteString("</ul></ol>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteString("\n")
               }
               state = stateDefault
            }
         }

      case stateUnorderedListNestedOL:
         if indent, ok := isCodeFence(line); ok {
            codeBlockIndent = indent
            codeBlockReturnState = stateUnorderedListNestedOL
            state = stateCodeBlock
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               result.WriteString(renderListLi(ordText))
            } else if isUnord && !isIndented(line) {
               result.WriteString("</ol>")
               result.WriteString(renderListLi(unordText))
               state = stateUnorderedList
            } else if isBlank(line) {
               result.WriteString("</ol>")
               state = stateUnorderedListPending
            } else {
               result.WriteString("</ol></ul>")
               if strings.TrimSpace(line) == "---" {
                  result.WriteString("<hr>\n")
               } else {
                  result.WriteString(renderInline(line))
                  result.WriteString("\n")
               }
               state = stateDefault
            }
         }

      case stateOrderedListPending:
         if isBlank(line) {
            result.WriteString("</ol>\n")
            state = stateDefault
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isOrd {
               result.WriteString(renderListLi(ordText))
               state = stateOrderedList
            } else if isUnord && isIndented(line) {
               result.WriteString("<ul>")
               result.WriteString(renderListLi(unordText))
               state = stateOrderedListNestedUL
            } else if isUnord {
               result.WriteString("</ol>")
               result.WriteString("<ul>")
               result.WriteString(renderListLi(unordText))
               state = stateUnorderedList
            } else if indent, ok := isCodeFence(line); ok {
               result.WriteString("</ol>")
               codeBlockIndent = indent
               codeBlockReturnState = stateDefault
               state = stateCodeBlock
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

      case stateUnorderedListPending:
         if isBlank(line) {
            result.WriteString("</ul>\n")
            state = stateDefault
         } else {
            ordText, isOrd := isOrderedListLine(line)
            unordText, isUnord := isUnorderedListLine(line)
            if isUnord {
               result.WriteString(renderListLi(unordText))
               state = stateUnorderedList
            } else if isOrd && isIndented(line) {
               result.WriteString("<ol>")
               result.WriteString(renderListLi(ordText))
               state = stateUnorderedListNestedOL
            } else if isOrd {
               result.WriteString("</ul>")
               result.WriteString("<ol>")
               result.WriteString(renderListLi(ordText))
               state = stateOrderedList
            } else if indent, ok := isCodeFence(line); ok {
               result.WriteString("</ul>")
               codeBlockIndent = indent
               codeBlockReturnState = stateDefault
               state = stateCodeBlock
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
      }
   }

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

   return result.String()
}
