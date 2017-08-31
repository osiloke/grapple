package scraper

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/apex/log"
)

// StringValFromCSSPath tries to get a string from a node
func StringValFromCSSPath(path []string, node *goquery.Selection) (string, bool) {
	var val string
	if len(path) == 1 {
		val = node.Text()
	} else {
		if v, ok := node.Attr(path[1]); ok {
			val = v
			if len(path) > 2 {
				reg := path[2]
				r, _ := regexp.Compile(reg)
				m := r.FindStringSubmatch(val)
				if len(m) > 1 {
					val = m[1]
				}
			}
		} else {
			return "", false
		}
	}
	val = stringMinifier(val)
	return removeInvalidUtf(val), true
}

// SchemaFromString creates a schema from a json string
func SchemaFromString(data string) *Schema {
	var schema Schema
	if err := json.Unmarshal([]byte(data), &schema); err != nil {
		log.WithError(err).Error("Unable to load schema")
		return nil
	}
	return &schema
}
func removeInvalidUtf(s string) string {
	if !utf8.ValidString(s) {
		v := make([]rune, 0, len(s))
		for i, r := range s {
			if r == utf8.RuneError {
				_, size := utf8.DecodeRuneInString(s[i:])
				if size == 1 {
					continue
				}
			}
			v = append(v, r)
		}
		return string(v)
	}
	return strings.TrimSpace(s)
}

//http://intogooglego.blogspot.com.ng/2015/05/day-6-string-minifier-remove-whitespaces.html
func stringMinifier(in string) (out string) {
	white := false
	for _, c := range in {
		if unicode.IsSpace(c) {
			if !white {
				out = out + " "
			}
			white = true
		} else {
			out = out + string(c)
			white = false
		}
	}
	return
}
