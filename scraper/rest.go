package scraper

import (
	"bytes"
	"errors"
	. "github.com/cstockton/go-conv"
	"net/url"
	"text/template"
)

var ErrNoNextUrl = errors.New("no next url")
var ErrNoRows = errors.New("no rows")
var ErrNoData = errors.New("no data")

type EchoData struct {
	data []byte
}

func (e *EchoData) Get(path string) interface{} {
	return e.data
}

//ParseData is used to convert raw data bytes into structured `Data`
func ParseData(val func(data []byte) (Data, error)) func(*RestScraper) error {
	return func(r *RestScraper) error {
		r.parseData = val
		return nil
	}
}

//NextUrl is a function which generates a new url to scrape next
func NextUrl(val func(lastUrl *url.URL, data Data) (string, error)) func(*RestScraper) error {
	return func(r *RestScraper) error {
		r.nextUrl = val
		return nil
	}
}

//RowCount returns a function which gets the total number of rows in the rest api
func RowCount(val func(data Data) (int, error)) func(*RestScraper) error {
	return func(r *RestScraper) error {
		r.rowCount = val
		return nil
	}
}

func ParseRow(val func(data interface{}) (interface{}, error)) func(*RestScraper) error {
	return func(r *RestScraper) error {
		r.parseRow = val
		return nil
	}
}

func GetRows(val func(data Data) ([]interface{}, error)) func(*RestScraper) error {
	return func(r *RestScraper) error {
		r.getRows = val
		return nil
	}
}

//rest scraper
type RestScraper struct {
	nextUrl      func(lastUrl *url.URL, data Data) (string, error)
	rowCount     func(data Data) (int, error)
	parseData    func(data []byte) (Data, error)
	getRows      func(data Data) ([]interface{}, error)
	parseRow     func(data interface{}) (interface{}, error)
	scrapeLimit  int
	scrapedCount int
	client       Client
}

func URL(u string) *url.URL {
	url, _ := url.Parse(u)
	return url
}
func (s *RestScraper) ScrapeUrl(url *url.URL) (data []byte, err error) {
	data, err = s.client.GetBytes(url.String())
	if err != nil {
		return
	}
	if data == nil {
		return nil, ErrNoData
	}
	s.scrapedCount++
	return
}

func (s *RestScraper) ParseData(data []byte) (Data, error) {
	return s.parseData(data)
}
func (s *RestScraper) NextUrl(lastUrl *url.URL, data Data) (string, error) {
	return s.nextUrl(lastUrl, data)
}

func (s *RestScraper) Rows(data Data) ([]interface{}, error) {
	return s.getRows(data)
}

func (s *RestScraper) Row(data interface{}) (interface{}, error) {
	return s.parseRow(data)
}

func (s *RestScraper) Total(data Data) (int, error) {
	return s.rowCount(data)
}

func NewRestScraper(client Client, options ...func(*RestScraper) error) *RestScraper {
	r := RestScraper{
		client:       client,
		scrapeLimit:  1,
		scrapedCount: 0,
		nextUrl: func(lastUrl *url.URL, data Data) (string, error) {
			return "", ErrNoNextUrl
		},
		parseRow: func(data interface{}) (interface{}, error) {
			return nil, ErrNoRows
		},
		getRows: func(data Data) ([]interface{}, error) {
			return nil, ErrNoRows
		},
		rowCount: func(data Data) (int, error) {
			return 0, ErrNoRows
		},
		parseData: func(data []byte) (Data, error) {
			return &EchoData{data}, nil
		},
	}
	for _, o := range options {
		o(&r)
	}
	return &r
}

func NewJSONRestScraper(client Client, nextUrlTemplate, rowCountPath, dataPath string) *RestScraper {
	t := template.New("nexturl")
	fmap := template.FuncMap{
		"param": func(url *url.URL, key string) string {
			return url.Query().Get(key)
		},
		"add": func(args ...interface{}) int {
			return Int(args[0]) + Int(args[1])
		},
		"setParams": func(_url *url.URL, args ...interface{}) *url.URL {
			// params := url.Values{}
			// for k, v := range _url.Query() {
			// 	params.Add(k, v[0])
			// }
			params := _url.Query()
			for i := 0; i < len(args); i += 2 {
				params.Set(String(args[i]), String(args[i+1]))
			}
			_url.RawQuery = params.Encode()
			return _url
		},
	}
	t.Funcs(fmap)
	tpl, err := t.Parse(nextUrlTemplate)
	if err != nil {
		panic(err)
	}
	return &RestScraper{
		client:      client,
		scrapeLimit: 1,
		nextUrl: func(lastUrl *url.URL, data Data) (string, error) {
			var _url bytes.Buffer
			if err := tpl.Execute(&_url, map[string]interface{}{
				"data": data,
				"url":  lastUrl,
			}); err != nil {
				return "", err
			}
			url := _url.String()
			if url == "" {
				return "", ErrNoNextUrl
			}
			return url, nil
		},
		parseRow: func(data interface{}) (interface{}, error) {
			return data, nil
		},
		getRows: func(data Data) ([]interface{}, error) {
			if v, ok := data.Get(dataPath).([]interface{}); ok {
				return v, nil
			}
			return nil, ErrNoData
		},
		rowCount: func(data Data) (int, error) {
			return int(data.Get(rowCountPath).(float64)), nil
		},
		parseData: func(data []byte) (Data, error) {
			return NewJSONData(data), nil
		},
	}
}
