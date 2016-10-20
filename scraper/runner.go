package scraper

// func OnNextRow(onNextRow func(row interface{})) func(*RestScraper) error {
// 	return func(r *RestScraper) error {
// 		r.onNextRow = onNextRow
// 		return nil
// 	}
// }

type Runner struct {
	url     string
	lastUrl string
	scraper Scraper
}

func (s *Runner) scrape(url string) (rows []interface{}, err error) {
	data, err := s.scraper.ScrapeUrl(s.url)
	if err != nil {
		return
	}
	rows, err = s.scraper.GetRows(data)
	if err != nil {
		return
	}
	for k, _row := range rows {
		row, _ := s.scraper.ParseRow(_row)
		rows[k] = row
	}
	return
}

func (s *Runner) Run() {

}
