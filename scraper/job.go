package scraper

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/codeskyblue/go-sh"
	"github.com/mgutz/logxi/v1"
	"github.com/paulbellamy/ratecounter"
	"github.com/ungerik/go-dry"
	//	"github.com/kennygrant/sanitize"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/robertkrimen/otto"
)

var logger = log.NewLogger(log.NewConcurrentWriter(os.Stdout), "scraper")

type TypeFormatter func(property *Schema, sel *goquery.Selection) interface{}

var CUSTOM_TYPES = map[string]TypeFormatter{}

func AddCustomType(name string, fn TypeFormatter) {
	CUSTOM_TYPES[name] = fn
}

type JobStats struct {
	TotalItems     ratecounter.Counter
	ProcessedItems ratecounter.Counter
}

type Schema struct {
	Id              string   `json:"id" description:"id of field"`
	CssPath         []string `json:"css" description:"relative css from parent schema"`
	NextPath        []string `json:"next" description:"path to next list of urls if this is a urllist"`
	Type            string   `json:"type" description:"type of schema, object, int, string etc"`
	MergeWithParent bool     `json:"mergeWithParent"`
	KeyPath         []string `json:"key"`
	ValPath         []string `json:"val"`
	Limit           int      `json:"limit"`
	Properties      []Schema `json:"properties" description:"sub schema"`
}

type Job struct {
	Name                 string
	URL                  string
	JobSchema            *Schema
	StopOnFn             StopOn
	Stats                *JobStats
	Doc                  *goquery.Document
	Con                  Client
	lastIp               string
	UniqueIp             bool
	ChildPageRequestRate time.Duration
}

type StopOn func(i int, item map[string]interface{}) bool

func cleanText(val string) string {
	return stringMinifier(removeInvalidUtf(val))
}

func (j *Job) getProperty(vm *otto.Otto, parentNode *goquery.Selection, property *Schema) interface{} {
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
	logger.Info("parent node found " + path)
	if propertyNode != nil {
		switch property.Type {
		case OBJECT_PROPERTY:
			var child_data = make(map[string]interface{})
			for _, child_property := range property.Properties {
				child_data[child_property.Id] = j.getProperty(vm, propertyNode, &child_property)
			}
			return child_data
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
			} else if val, ok := StringValFromCSSPath(property.CssPath, propertyNode); ok {
				return val
			}
			return nil
		}
	}
	logger.Warn("Unable to find property in document", "id", property.Id, "css", property.CssPath[0])
	return nil
}
func (j *Job) Do() chan map[string]interface{} {
	finished := make(chan map[string]interface{})
	j.Stats = &JobStats{0, 0}
	stats := j.Stats

	if j.JobSchema == nil {
		logger.Error("Schema is not available")
		close(finished)
		return finished
	}
	vm := otto.New()
	// doc, err := goquery.NewDocument(j.URL)
	if ipB, err := j.Con.GetBytes("http://ifconfig.me"); err == nil {
		logger.Debug("Using ip from tor proxy", "ip", string(ipB))
	} else {
		logger.Info("Unable to retrieve ifconfig", "err", err.Error())
	}
	doc, err := j.Con.GetDoc(j.URL)
	if err != nil {
		logger.Fatal("Could not retrieve url", "err", err, "url", j.URL)
		close(finished)
		return finished
	}
	j.Doc = doc
	logger.Debug("Retrieved document from url", "url", j.URL, "doc", doc.Length())

	go func() {
		doc.Find(j.JobSchema.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
			logger.Info(fmt.Sprintf("Item %d", i))
			var data = make(map[string]interface{})
			for _, property := range j.JobSchema.Properties {
				val := j.getProperty(vm, s, &property)
				data[property.Id] = val
			}
			if j.StopOnFn != nil {
				if j.StopOnFn(i, data) {
					return false
				}
			}
			finished <- data
			stats.TotalItems++
			return true
		})
		close(finished)
	}()
	return finished
}

func (j *Job) DoSave() *JobStats {
	stats := &JobStats{0, 0}

	if j.JobSchema == nil {
		logger.Error("Schema is not available")
		return stats
	}
	vm := otto.New()
	// doc, err := goquery.NewDocument(j.URL)
	if ipB, err := j.Con.GetBytes("https://api.ipify.org"); err == nil {
		_ip := string(ipB)
		if j.UniqueIp && strings.Contains(_ip, j.lastIp) && j.Con.SocksEnabled() {
			//send sighup to tor before retrying
			if runtime.GOOS == "linux" {
				log.Info("restarting tor")
				sh.Command("service", "tor", "restart").Run()
				time.Sleep(time.Second * 10)
			}
		}
		j.lastIp = _ip
		logger.Debug("Using ip from tor proxy", "ip", _ip)
	} else {
		logger.Info("Unable to retrieve ifconfig", "err", err.Error())
	}
	// TODO: Handle http error codes properly for 400, 429
	doc, err := j.Con.GetDoc(j.URL)
	if err != nil {
		logger.Warn("Could not retrieve url", "err", err, "url", j.URL)
		return stats
	}
	logger.Debug("Retrieved document from url", "url", j.URL, "doc", doc.Length())
	finished := make(chan map[string]interface{})
	done := make(chan bool)

	filename := j.Name + ".json"

	logger.Info("Saving to " + filename)

	//Async filewriter
	go func(in chan map[string]interface{}, done chan<- bool) {
		for item := range in {
			if len(item) > 0 {
				if data, err := json.Marshal(item); err != nil {
					logger.Error("Unable to save entry", "err", err)
				} else {
					dry.FileAppendBytes(filename, data)
					dry.FileAppendBytes(filename, []byte("\n"))
				}
				stats.ProcessedItems.Incr(1)
			}
		}
		done <- true
		logger.Info(fmt.Sprintf("Processed %d/%d items", stats.ProcessedItems.Value(), stats.TotalItems.Value()))
	}(finished, done)

	if j.JobSchema.Type == URLLIST_PROPERTY {
		logger.Info("Retrieving list of urls to scrape from " + j.URL)
		limit := j.JobSchema.Limit
		count := 1
		doc.Find(j.JobSchema.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
			//get list element url page
			//TODO: check if url is relative or absolute
			//Exponential backoff using go backoff
			if url, ok := StringValFromCSSPath(j.JobSchema.CssPath, s); ok {
				url = removeInvalidUtf(stringMinifier(url))
				url = strings.TrimPrefix(url, doc.Url.String())
				url = doc.Url.Scheme + "://" + doc.Url.Host + "/" + strings.TrimPrefix(url, "/")
				childDoc, err := j.Con.GetDoc(url)
				if err != nil {
					logger.Fatal("Could not retrieve child url", "err", err, "url", url)
					return false
				}

				logger.Info("=== parsing data from url " + url)
				for _, property := range j.JobSchema.Properties {
					log.Info("find csspath " + property.CssPath[0])
					childDoc.Find(property.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
						// html, _ := s.Html()
						// log.Info("=== found " + html)
						var data = make(map[string]interface{})
						for _, nestedProperty := range property.Properties {
							val := j.getProperty(vm, s, &nestedProperty)
							if nestedProperty.MergeWithParent == true {
								for k, v := range val.(map[string]interface{}) {
									data[k] = v
								}
							} else {
								if val != nil {
									data[nestedProperty.Id] = val
								}
							}
						}
						if j.StopOnFn(i, data) {
							return false
						}
						finished <- data
						stats.TotalItems.Incr(1)
						return true
					})
				}
				if count == limit {
					log.Info(fmt.Sprintf("%d/%d child urls processed", count, limit))
					return false
				}
				count = count + 1
				return true
			}
			log.Warn("child url css path failed", "path", j.JobSchema.CssPath)
			return false
		})
	} else {
		doc.Find(j.JobSchema.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
			var data = make(map[string]interface{})
			for _, property := range j.JobSchema.Properties {
				val := j.getProperty(vm, s, &property)
				data[property.Id] = val
			}
			if j.StopOnFn(i, data) {
				return false
			}
			finished <- data
			stats.TotalItems.Incr(1)
			return true
		})
	}
	close(finished)
	<-done
	close(done)

	logger.Info("Completed: " + filename)
	return stats
}

func (j *Job) ScrapeStream() (chan map[string]interface{}, error) {
	if j.JobSchema == nil {
		return nil, ErrNoSchema
	}
	vm := otto.New()
	// doc, err := goquery.NewDocument(j.URL)
	if ipB, err := j.Con.GetBytes("https://api.ipify.org"); err == nil {
		_ip := string(ipB)
		if j.UniqueIp && strings.Contains(_ip, j.lastIp) && j.Con.SocksEnabled() {
			//send sighup to tor before retrying
			if runtime.GOOS == "linux" {
				log.Info("restarting tor")
				sh.Command("service", "tor", "restart").Run()
				time.Sleep(time.Second * 10)
			}
		}
		j.lastIp = _ip
		logger.Debug("Using ip from tor proxy", "ip", _ip)
	} else {
		logger.Info("Unable to retrieve ifconfig", "err", err.Error())
	}
	// TODO: Handle http error codes properly for 400, 429
	doc, err := j.Con.GetDoc(j.URL)
	if err != nil {
		logger.Warn("Could not retrieve url", "err", err, "url", j.URL)
		return nil, err
	}
	logger.Debug("Retrieved document from url", "url", j.URL, "doc", doc.Length())
	rows := make(chan map[string]interface{})
	go func(rows chan map[string]interface{}) {
		defer close(rows)
		if j.JobSchema.Type == URLLIST_PROPERTY {
			logger.Info("Retrieving list of urls to scrape from " + j.URL)
			limit := j.JobSchema.Limit
			count := 1
			doc.Find(j.JobSchema.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
				//get list element url page
				//TODO: check if url is relative or absolute
				//Exponential backoff using go backoff
				if url, ok := StringValFromCSSPath(j.JobSchema.CssPath, s); ok {
					url = removeInvalidUtf(stringMinifier(url))
					url = strings.TrimPrefix(url, doc.Url.String())
					url = doc.Url.Scheme + "://" + doc.Url.Host + "/" + strings.TrimPrefix(url, "/")
					childDoc, err := j.Con.GetDoc(url)
					if err != nil {
						logger.Fatal("Could not retrieve child url", "err", err, "url", url)
						return false
					}

					logger.Info("=== parsing data from url " + url)
					for _, property := range j.JobSchema.Properties {
						log.Info("find csspath " + property.CssPath[0])
						childDoc.Find(property.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
							// html, _ := s.Html()
							// log.Info("=== found " + html)
							var data = make(map[string]interface{})
							for _, nestedProperty := range property.Properties {
								val := j.getProperty(vm, s, &nestedProperty)
								if nestedProperty.MergeWithParent == true {
									for k, v := range val.(map[string]interface{}) {
										data[k] = v
									}
								} else {
									if val != nil {
										data[nestedProperty.Id] = val
									}
								}
							}
							rows <- data
							return true
						})
					}
					if count == limit {
						log.Info(fmt.Sprintf("%d/%d child urls processed", count, limit))
						return false
					}
					count = count + 1
					return true
				}
				log.Warn("child url css path failed", "path", j.JobSchema.CssPath)
				return false
			})
		} else {
			doc.Find(j.JobSchema.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
				var data = make(map[string]interface{})
				for _, property := range j.JobSchema.Properties {
					val := j.getProperty(vm, s, &property)
					data[property.Id] = val
				}
				rows <- data
				return true
			})
		}
	}(rows)
	return rows, nil
}
func (j *Job) ScrapeInputStream(input chan string) (chan map[string]interface{}, error) {
	if j.JobSchema == nil {
		return nil, ErrNoSchema
	}
	vm := otto.New()
	// doc, err := goquery.NewDocument(j.URL)
	if ipB, err := j.Con.GetBytes("https://api.ipify.org"); err == nil {
		_ip := string(ipB)
		if j.UniqueIp && strings.Contains(_ip, j.lastIp) && j.Con.SocksEnabled() {
			//send sighup to tor before retrying
			if runtime.GOOS == "linux" {
				log.Info("restarting tor")
				sh.Command("service", "tor", "restart").Run()
				time.Sleep(time.Second * 10)
			}
		}
		j.lastIp = _ip
		logger.Debug("Using ip from tor proxy", "ip", _ip)
	} else {
		logger.Info("Unable to retrieve ifconfig", "err", err.Error())
	}
	// TODO: Handle http error codes properly for 400, 429
	doc, err := j.Con.GetDoc(j.URL)
	if err != nil {
		logger.Warn("Could not retrieve url", "err", err, "url", j.URL)
		return nil, err
	}
	logger.Debug("Retrieved document from url", "url", j.URL, "doc", doc.Length())
	rows := make(chan map[string]interface{})
	go func(rows chan map[string]interface{}) {
		defer close(rows)
		if j.JobSchema.Type == URLLIST_PROPERTY {
			logger.Info("Retrieving list of urls to scrape from " + j.URL)
			limit := j.JobSchema.Limit
			count := 1
			doc.Find(j.JobSchema.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
				//get list element url page
				//TODO: check if url is relative or absolute
				//Exponential backoff using go backoff
				if url, ok := StringValFromCSSPath(j.JobSchema.CssPath, s); ok {
					url = removeInvalidUtf(stringMinifier(url))
					url = strings.TrimPrefix(url, doc.Url.String())
					url = doc.Url.Scheme + "://" + doc.Url.Host + "/" + strings.TrimPrefix(url, "/")
					childDoc, err := j.Con.GetDoc(url)
					if err != nil {
						logger.Fatal("Could not retrieve child url", "err", err, "url", url)
						return false
					}

					logger.Info("=== parsing data from url " + url)
					for _, property := range j.JobSchema.Properties {
						log.Info("find csspath " + property.CssPath[0])
						childDoc.Find(property.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
							// html, _ := s.Html()
							// log.Info("=== found " + html)
							var data = make(map[string]interface{})
							for _, nestedProperty := range property.Properties {
								val := j.getProperty(vm, s, &nestedProperty)
								if nestedProperty.MergeWithParent == true {
									for k, v := range val.(map[string]interface{}) {
										data[k] = v
									}
								} else {
									if val != nil {
										data[nestedProperty.Id] = val
									}
								}
							}
							rows <- data
							return true
						})
					}
					if count == limit {
						log.Info(fmt.Sprintf("%d/%d child urls processed", count, limit))
						return false
					}
					count = count + 1
					return true
				}
				log.Warn("child url css path failed", "path", j.JobSchema.CssPath)
				return false
			})
		} else {
			doc.Find(j.JobSchema.CssPath[0]).EachWithBreak(func(i int, s *goquery.Selection) bool {
				var data = make(map[string]interface{})
				for _, property := range j.JobSchema.Properties {
					val := j.getProperty(vm, s, &property)
					data[property.Id] = val
				}
				rows <- data
				return true
			})
		}
	}(rows)
	return rows, nil
}
