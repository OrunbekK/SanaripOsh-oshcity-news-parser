package scraper

type Card struct {
	Title       string
	URL         string
	DateRaw     string
	SequenceNum int // Номер в листинге (для сортировки)
}

type Selectors struct {
	ListContainer  string   `yaml:"list_container"`
	CardSelectors  string   `yaml:"card_selectors"`
	TitleSelectors []string `yaml:"title_selectors"`
	URLSelectors   []string `yaml:"url_selectors"`
	DateSelectors  []string `yaml:"date_selectors"`
	NextPageLink   []string `yaml:"next_page_link"`
}
