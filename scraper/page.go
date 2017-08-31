package scraper

import (
	"bytes"
	"errors"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/apex/log"
	"github.com/robertkrimen/otto"
	dry "github.com/ungerik/go-dry"
)

// PageOpt page scraper options
type PageOpt func(p *PageScraper) *PageScraper

// PageRequest creates a request for a page
type PageRequest struct {
	URL    string
	Method string
	Params url.Values
}

var defaultRequestGetter = func(path string) *PageRequest {
	return &PageRequest{URL: path, Method: "GET"}
}

// PageScraper scrapes an html page
type PageScraper struct {
	schema        *Schema
	con           Client
	requestGetter func(path string) *PageRequest
}

func (p *PageScraper) getProperty(vm *otto.Otto, parentNode *goquery.Selection, property *Schema) interface{} {
	if parentNode == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Unable to compile css path", "css", property.CssPath[0], "err", r)
		}
	}()
	var propertyNode *goquery.Selection
	path := property.CssPath[0]
	if path == "" {
		propertyNode = parentNode
	} else {
		propertyNode = parentNode.Find(path)
	}
	if propertyNode != nil {
		switch property.Type {
		case OBJECT_PROPERTY:
			var childData = make(map[string]interface{})
			for _, childProperty := range property.Properties {
				childData[childProperty.Id] = p.getProperty(vm, propertyNode, &childProperty)
			}
			return childData
		case LONGTEXT_PROPERTY:
			if len(property.CssPath) > 1 {
				if val, ok := propertyNode.Attr(property.CssPath[1]); ok {
					val = stringMinifier(val)
					return removeInvalidUtf(val)
				}
				return nil
			}
			return removeInvalidUtf(propertyNode.Text())
		case IMAGE_PROPERTY:
			if val, ok := StringValFromCSSPath(property.CssPath, propertyNode); ok {
				return val
			}
			return nil
		case ARRAY_PROPERTY:
			var result []string
			if len(property.CssPath) > 1 {
				propertyNode.Each(func(i int, s *goquery.Selection) {
					if val, ok := s.Attr(property.CssPath[1]); ok {
						result = append(result, cleanText(val))
					}
				})
				// for _, v := range propertyNode.Nodes {
				// 	if val, ok := v.Attr(property.CssPath[1]); ok {
				// 		result = append(result, cleanText(val))
				// 	}
				// }
			} else {
				propertyNode.Each(func(i int, s *goquery.Selection) {
					result = append(result, cleanText(s.Text()))
				})
				// for _, v := range propertyNode.Nodes {
				// 	result = append(result, cleanText(v.Data))
				// }
			}
			return result
		case STRING_PROPERTY:
			if val, ok := StringValFromCSSPath(property.CssPath, propertyNode); ok {
				return val
			}
			return nil
		case INT_PROPERTY:
			if len(property.CssPath) > 1 {
				if val, ok := propertyNode.Attr(property.CssPath[1]); ok {
					return dry.StringToInt(removeInvalidUtf(val))
				}
				return nil
			}
			return dry.StringToInt(removeInvalidUtf(propertyNode.Text()))
		case PROPERTY_ARRAY:
			props := map[string]interface{}{}
			//an array of properties with key and value combinations
			propertyNode.EachWithBreak(func(i int, s *goquery.Selection) bool {
				//retrieves all items with the same css path then gets the key-value pair
				//as specified in the property definition
				//get key
				var (
					key string
					ok  bool
				)
				if property.KeyPath[0] == "" {

					key, ok = StringValFromCSSPath(property.KeyPath, s.Clone().Children().Remove().End())

				} else {
					key, ok = StringValFromCSSPath(property.KeyPath, s.Find(property.KeyPath[0]).First())
				}
				if len(key) == 0 || !ok {
					return true
				}
				key = strings.ToLower(strings.Trim(dry.StringReplaceMulti(key, ",", "", ".", "", ":", " ", " ", "_"), "_"))
				// key = dry.StringToLowerCamelCase(key)
				//get val
				val, ok := StringValFromCSSPath(property.ValPath, s.Find(property.ValPath[0]).First())
				if !ok {
					return true
				}
				if strings.ToLower(val) == "nil" {
					return true
				}
				if len(val) > 0 {
					props[key] = val
				}
				return true
			})
			return props
		default:
			//check type formatters
			if customType, ok := CUSTOM_TYPES[property.Type]; ok {
				return customType(property, propertyNode)
			}
			val, _ := StringValFromCSSPath(property.CssPath, propertyNode)
			return val
		}
	}
	log.WithFields(log.Fields{
		"id":  property.Id,
		"css": property.CssPath,
	}).Warn("Unable to find property in document")
	return nil
}

// ScrapeURL get data from one url
func (p *PageScraper) ScrapeURL(url string) ([]byte, error) {
	req := p.requestGetter(url)
	if req.Method == "POST" {
		return p.con.PostBytes(req.URL, req.Params)
	}
	return p.con.GetBytes(req.URL)
}

// GetNextURL get rows using goquery
func (p *PageScraper) GetNextURL(lastURL string, data []byte) (string, error) {
	return "", errors.New("not implemented")
}

// GetRows get rows using goquery
func (p *PageScraper) GetRows(data []byte) ([]interface{}, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err == nil {
		rows := []interface{}{}

		doc.Find(p.schema.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
			var data = make(map[string]interface{})
			for _, property := range p.schema.Properties {
				val := p.getProperty(nil, s, &property)
				data[property.Id] = val
			}
			rows = append(rows, data)
			return true
		})
		return rows, nil
	}
	return nil, err

}

// ParseRow parse a single row
func (p *PageScraper) ParseRow(data interface{}) (interface{}, error) {
	return nil, errors.New("not implemented")
}

// SetRequestGetter sets a request getter method for the scraper
func SetRequestGetter(rg func(path string) *PageRequest) func(p *PageScraper) *PageScraper {
	return func(p *PageScraper) *PageScraper {
		p.requestGetter = rg
		return p
	}
}

// NewPageScraper returns a new PageScraper
func NewPageScraper(con Client, schema *Schema, opts ...PageOpt) *PageScraper {
	p := &PageScraper{schema: schema, con: con}
	for _, opt := range opts {
		opt(p)
	}
	return p
}
