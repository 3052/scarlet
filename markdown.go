// markdown.go
package scarlet

import (
   "html"
   "strings"
)

func escapeHTML(s string) string {
   return strings.ReplaceAll(html.EscapeString(s), "\n", "<br>\n")
}
