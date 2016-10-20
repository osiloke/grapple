package scraper

import (
	"github.com/Jeffail/gabs"
)

type JSONData struct {
	data *gabs.Container
}

func (j *JSONData) Get(path string) interface{} {
	return j.data.Path(path).Data()
}

func NewJSONData(data []byte) *JSONData {
	jsonParsed, _ := gabs.ParseJSON(data)
	return &JSONData{jsonParsed}
}
