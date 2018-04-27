package scraper

type Scraper interface {
	ScrapeURL(string) ([]byte, error)
	GetNextURL(lastUtl string, data []byte) (string, error)
	GetRows(data []byte) ([]interface{}, error)
	ParseRow(data interface{}) (interface{}, error)
}

type Data interface {
	Get(string) interface{}
}

//a rest api scraper
