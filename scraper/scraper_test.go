package scraper

import (
	"bytes"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/osiloke/grapple/mocks"
	. "github.com/smartystreets/goconvey/convey"
	. "github.com/stretchr/testify/mock"
)

var testHtml = `
<html>
<body>
<div>
</div>
<table id="customers">
  <tbody>
  <tr>
    <th>Company</th>
    <th>Contact</th>
    <th>Country</th>
  </tr>
  <tr>
    <td>Alfreds Futterkiste</td>
    <td>Maria Anders</td>
    <td>Germany</td>
  </tr>
  <tr>
    <td>Centro comercial Moctezuma</td>
    <td>Francisco Chang</td>
    <td>Mexico</td>
  </tr>
  <tr>
    <td>Ernst Handel</td>
    <td>Roland Mendel</td>
    <td>Austria</td>
  </tr>
  <tr>
    <td>Island Trading</td>
    <td>Helen Bennett</td>
    <td>UK</td>
  </tr>
  <tr>
    <td>Laughing Bacchus Winecellars</td>
    <td>Yoshi Tannamuri</td>
    <td>Canada</td>
  </tr>
  <tr>
    <td>Magazzini Alimentari Riuniti</td>
    <td>Giovanni Rovelli</td>
    <td>Italy</td>
  </tr>
</tbody></table> 
</body>
</html>
`

func TestScrapeStreamTable(t *testing.T) {
	r := bytes.NewBufferString(testHtml)
	doc, _ := goquery.NewDocumentFromReader(r)
	client := mocks.Client{}
	client.On("GetBytes", AnythingOfType("string")).Return([]byte(testHtml), nil)
	client.On("GetDoc", AnythingOfType("string")).Return(doc, nil)
	SkipConvey("create a job", t, func() {

		var testSchema = SchemaFromString(`
{
	"name": "example schema",
	"id": "example",
	"type": "object",
	"css": ["table tr"],
	"properties": [ 
	     {
	     	"id": "company",
	     	"type": "string",
	     	"css": ["td:nth-of-type(1)"]
	     },
	     {
	     	"id": "contact",
	     	"type": "string",
	     	"css": ["td:nth-of-type(2)"]
	     },
	     {
	     	"id": "country",
	     	"type": "string",
	     	"css": ["td:nth-of-type(3)"]
	     } 
	]

}
`)
		job := Job{
			Name:      "example scraper",
			URL:       "http://example.com",
			JobSchema: testSchema,
			StopOnFn:  nil,
			Con:       &client,
		}
		Convey("and scrape a web page", func() {
			rows, err := job.ScrapeStream()
			if err != nil {
				panic(err)
			}
			Convey("wait for all rows to be scraped", func() {
				for r := range rows {
					Println(r)
				}
			})
		})

	})
}
