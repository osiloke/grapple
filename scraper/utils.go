package scraper

import (
	"encoding/json"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"
)

func SchemaFromString(schema string) *Schema {
	var job_schema Schema
	if err := json.Unmarshal([]byte(schema), &job_schema); err != nil {
		log.Println("Unable to load schema", "err", err)
		return nil
	}
	return &job_schema
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
