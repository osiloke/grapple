package scraper

import (
	"encoding/json"
	"log"
)

const (
	STRING_PROPERTY   = "string"
	ARRAY_PROPERTY    = "array"
	URLLIST_PROPERTY  = "urllist"
	IMAGE_PROPERTY    = "imageurl"
	LONGTEXT_PROPERTY = "longtext"
	OBJECT_PROPERTY   = "object"
	INT_PROPERTY      = "integer"
	//PROPERTY_ARRAY of properties which can be merged with the parent property
	PROPERTY_ARRAY = "property_array"
	//KV_PROPERTY is special, it parses a path as a map[string]interface{} and merges it to its parent data
	KV_PROPERTY = "kv"
)

type parserFn func(v interface{}) (interface{}, error)

func Parser(name string, parser parserFn) func(s *ParserSchema) {
	return func(s *ParserSchema) {
		s.parsers[name] = parser
	}
}

type ParserSchema struct {
	Id      string         `json:"id" description:"id of field"`
	Path    []string       `json:"path" description:"relative path from parent schema"`
	Type    string         `json:"type" description:"type of schema, object, int, string etc"`
	Key     []string       `json:"key"`
	Val     []string       `json:"val"`
	Limit   int            `json:"limit"`
	Fields  []ParserSchema `json:"fields" description:"sub schema"`
	parsers map[string]parserFn
}

func parserSchemaFromString(data string) *ParserSchema {
	var s ParserSchema
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		log.Println("Unable to load schema", "err", err)
		return nil
	}
	return &s
}

func NewParserSchemaFromString(data string, parsers ...parserFn) *ParserSchema {
	s := parserSchemaFromString(data)
	for _, p := range parsers {
		p(s)
	}
	return s
}
