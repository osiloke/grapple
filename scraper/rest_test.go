package scraper

import (
	"errors"
	"git.progwebtech.com/osiloke/grapple/scraper/mocks"
	. "github.com/smartystreets/goconvey/convey"
	. "github.com/stretchr/testify/mock"
	"net/url"
	"testing"
)

func TestClientError(t *testing.T) {
	failed := errors.New("failed")
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(nil, failed)
	Convey("create a rest scraper", t, func() {
		s := NewRestScraper(&client)
		Convey("then scrape url that fails", func() {
			_, err := s.ScrapeUrl(URL("http://example.com/v1/json"))
			Convey("this should return an err", func() {
				So(err, ShouldEqual, failed)
			})
		})
	})
}

func TestScrapeUrlWithEmptyData(t *testing.T) {
	// restData := []string{`
	// 	{}
	// `}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(nil, nil)
	Convey("create a rest scraper", t, func() {
		s := NewRestScraper(&client)
		Convey("then scrape url that has no data", func() {
			_, err := s.ScrapeUrl(URL("http://example.com/v1/json"))
			Convey("this should return an ErrNoData", func() {
				So(err, ShouldEqual, ErrNoData)
			})
		})
	})
}

func TestScrapeUrlWithData(t *testing.T) {
	restData := `{"name":"val"}`
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return([]byte(restData), nil)
	Convey("create a rest scraper", t, func() {
		s := NewRestScraper(&client)
		Convey("then scrape url that has data", func() {
			data, _ := s.ScrapeUrl(URL("http://example.com/v1/json"))
			Convey("this should return some data", func() {
				So(string(data), ShouldEqual, restData)
			})
		})
	})
}

func TestNoNextUrl(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) ([]byte, error) {
		return []byte(restPages[url]), nil
	})
	Convey("create a rest scraper", t, func() {
		s := NewRestScraper(&client)
		Convey("then get next url ", func() {
			_, err := s.NextUrl(nil, nil)
			Convey("then get next url", func() {
				So(err, ShouldEqual, ErrNoNextUrl)
			})
		})
	})
}

func TestNextUrl(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) ([]byte, error) {
		return []byte(restPages[url]), nil
	})
	Convey("create a rest scraper", t, func() {
		s := NewRestScraper(
			&client,
			NextUrl(func(lastUrl *url.URL, data Data) (string, error) {
				return "http://example.com/v1/json?p=2", nil
			}),
		)
		Convey("then get next url ", func() {
			nextUrl, _ := s.NextUrl(nil, nil)
			Convey("then get next url", func() {
				So(nextUrl, ShouldEqual, "http://example.com/v1/json?p=2")
			})
		})
	})
}

func TestParseData(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) []byte {
		return []byte(restPages[url])
	}, nil)
	Convey("create a rest scraper with row count option", t, func() {
		s := NewRestScraper(
			&client,
			ParseData(func(data []byte) (Data, error) {
				return NewJSONData(data), nil
			}),
		)
		Convey("scrape url", func() {
			data, _ := s.ScrapeUrl(URL("http://example.com/v1/json"))
			Convey("then parse data", func() {
				edata, _ := s.ParseData(data)
				Convey("then parsed data should be an instance of JSONData", func() {
					jdata, _ := edata.(*JSONData)
					So(jdata, ShouldNotBeNil)
				})
			})
		})
	})
}

func TestDefaultRowCount(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) []byte {
		return []byte(restPages[url])
	}, nil)
	Convey("create a rest scraper with row count option", t, func() {
		s := NewRestScraper(
			&client)
		Convey("scrape url", func() {
			_, err := s.Total(nil)
			Convey("then parse data", func() {
				So(err, ShouldEqual, ErrNoRows)
			})
		})
	})
}
func TestDefaultGetRow(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) []byte {
		return []byte(restPages[url])
	}, nil)
	Convey("create a rest scraper with row count option", t, func() {
		s := NewRestScraper(
			&client)
		Convey("scrape url", func() {
			_, err := s.Row(nil)
			Convey("then parse data", func() {
				So(err, ShouldEqual, ErrNoRows)
			})
		})
	})
}
func TestDefaultParseRows(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) []byte {
		return []byte(restPages[url])
	}, nil)
	Convey("create a rest scraper with row count option", t, func() {
		s := NewRestScraper(
			&client)
		Convey("scrape url", func() {
			_, err := s.Rows(nil)
			Convey("then parse data", func() {
				So(err, ShouldEqual, ErrNoRows)
			})
		})
	})
}
func TestRowCount(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) []byte {
		return []byte(restPages[url])
	}, nil)
	Convey("create a rest scraper with row count option", t, func() {
		s := NewRestScraper(
			&client,
			RowCount(func(data Data) (int, error) {
				return 3, nil
			}),
		)
		Convey("scrape url", func() {
			data, _ := s.ScrapeUrl(URL("http://example.com/v1/json"))
			Convey("then parse data", func() {
				edata, _ := s.ParseData(data)
				Convey("then get total count from data", func() {
					total, _ := s.Total(edata)
					So(total, ShouldEqual, 3)
				})
			})
		})
	})
}

func TestGetRows(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) []byte {
		return []byte(restPages[url])
	}, nil)
	Convey("create a rest scraper with row count option", t, func() {
		s := NewRestScraper(
			&client,
			RowCount(func(data Data) (int, error) {
				return int(data.Get("total").(float64)), nil
			}),
			ParseData(func(data []byte) (Data, error) {
				return NewJSONData(data), nil
			}),
			GetRows(func(data Data) ([]interface{}, error) {
				return data.Get("data").([]interface{}), nil
			}),
		)
		Convey("scrape url", func() {
			data, _ := s.ScrapeUrl(URL("http://example.com/v1/json"))
			Convey("then parse data", func() {
				edata, _ := s.ParseData(data)
				Convey("then get total count from data", func() {
					total, _ := s.Total(edata)
					So(total, ShouldEqual, 3)
				})
			})
		})
	})
}

func TestParseRow(t *testing.T) {
	restPages := map[string]string{
		"http://example.com/v1/json":     `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		"http://example.com/v1/json?p=2": `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) []byte {
		return []byte(restPages[url])
	}, nil)
	Convey("create a rest scraper with row count option", t, func() {
		s := NewRestScraper(
			&client,
			RowCount(func(data Data) (int, error) {
				return int(data.Get("total").(float64)), nil
			}),
			ParseData(func(data []byte) (Data, error) {
				return NewJSONData(data), nil
			}),
			GetRows(func(data Data) ([]interface{}, error) {
				return data.Get("data").([]interface{}), nil
			}),
			ParseRow(func(row interface{}) (interface{}, error) {
				return row, nil
			}),
		)
		Convey("scrape url", func() {
			data, _ := s.ScrapeUrl(URL("http://example.com/v1/json"))
			Convey("then parse data", func() {
				edata, _ := s.ParseData(data)
				rows, _ := s.Rows(edata)
				Convey("then get total count from data", func() {
					row, _ := s.Row(rows[0])
					So(row.(map[string]interface{}), ShouldResemble, edata.Get("data").([]interface{})[0])
				})
			})
		})
	})
}

func TestEchoDataGet(t *testing.T) {
	Convey("given echo data", t, func() {
		e := EchoData{[]byte("daa")}
		Convey("calling get on any path should return original data", func() {
			So(string(e.Get("something").([]byte)), ShouldEqual, string(e.data))
		})
	})
}

func TestJSONRestScraperNextUrl(t *testing.T) {
	url1 := "http://example.com/v1/json?p=1"
	url2 := "http://example.com/v1/json?p=2"
	restPages := map[string]string{
		url1: `{"total":3, "data":[{"name":"one"},{"name":"two"},{"name":"three"}]}`,
		url2: `{"total":3, "data":[{"name":"four"},{"name":"five"},{"name":"siz"}]}`,
	}
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return(func(url string) []byte {
		return []byte(restPages[url])
	}, nil)
	Convey("create a rest scraper", t, func() {
		s := NewJSONRestScraper(
			&client,
			`{{$p := .url.Query.Get "p" }}{{$next := add $p 1}}{{setParams .url "p" $next}}`,
			"total_product_count",
			"products",
		)
		url, _ := url.Parse(url1)
		Convey("scrape a url", func() {

			data, _ := s.ScrapeUrl(URL(url1))
			Convey("parse data retrieved", func() {
				edata, _ := s.ParseData(data)
				Convey("then get next url ", func() {
					nextUrl, err := s.NextUrl(url, edata)
					if err != nil {
						Println(err)
					}
					Convey("then get next url", func() {
						So(nextUrl, ShouldEqual, url2)
					})
				})
			})
		})
	})
}
