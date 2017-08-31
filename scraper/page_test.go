package scraper

import (
	"reflect"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
)

var client Client

func init() {

	client, _ = NewDefaultClient(nil)
}

var exampleCom = `<!doctypehtml>
<html>
<head>
<title>ExampleDomain</title>

<metacharset="utf-8"/>
<metahttp-equiv="Content-type"content="text/html;charset=utf-8"/>
<metaname="viewport"content="width=device-width,initial-scale=1"/>
<styletype="text/css">
body{
background-color:#f0f0f2;
margin:0;
padding:0;
font-family:"OpenSans","HelveticaNeue",Helvetica,Arial,sans-serif;

}
div{
width:600px;
margin:5emauto;
padding:50px;
background-color:#fff;
border-radius:1em;
}
a:link,a:visited{
color:#38488f;
text-decoration:none;
}
@media(max-width:700px){
body{
background-color:#fff;
}
div{
width:auto;
margin:0auto;
border-radius:0;
padding:1em;
}
}
</style>
</head>

<body>
<div>
<h1>ExampleDomain</h1>
<p>Thisdomainisestablishedtobeusedforillustrativeexamplesindocuments.Youmayusethis
domaininexampleswithoutpriorcoordinationoraskingforpermission.</p>
<p><ahref="http://www.iana.org/domains/example">Moreinformation...</a></p>
</div>
</body>
</html>`

func TestPageScraper_getProperty(t *testing.T) {
	type args struct {
		vm         *otto.Otto
		parentNode *goquery.Selection
		property   *Schema
	}
	tests := []struct {
		name string
		p    *PageScraper
		args args
		want interface{}
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.getProperty(tt.args.vm, tt.args.parentNode, tt.args.property); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PageScraper.getProperty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPageScraper_ScrapeURL(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		p       *PageScraper
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Test get example.com",
			p:    NewPageScraper(client, nil),
			args: args{url: "http://example.com"},
			want: []byte(exampleCom),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.p.ScrapeURL(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("PageScraper.ScrapeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got = []byte(strings.Replace(strings.TrimSpace(string(got)), " ", "", -1))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PageScraper.ScrapeURL() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func TestPageScraper_GetNextURL(t *testing.T) {
	type args struct {
		lastURL string
		data    []byte
	}
	tests := []struct {
		name    string
		p       *PageScraper
		args    args
		want    string
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.p.GetNextURL(tt.args.lastURL, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("PageScraper.GetNextURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PageScraper.GetNextURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPageScraper_GetRows(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		p       *PageScraper
		args    args
		want    []interface{}
		wantErr bool
	}{
		{
			name: "Test get example.com",
			p: NewPageScraper(
				client,
				SchemaFromString(`{
					"name": "example title",
					"css": ["body"],
					"properties": [{
						"id":"title",
						"css": ["h1"]
					}]
			 }`)),
			args: args{[]byte(exampleCom)},
			want: []interface{}{map[string]interface{}{
				"title": "ExampleDomain",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.p.GetRows(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("PageScraper.GetRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PageScraper.GetRows() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPageScraper_ParseRow(t *testing.T) {
	type args struct {
		data interface{}
	}
	tests := []struct {
		name    string
		p       *PageScraper
		args    args
		want    interface{}
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.p.ParseRow(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("PageScraper.ParseRow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PageScraper.ParseRow() = %v, want %v", got, tt.want)
			}
		})
	}
}
