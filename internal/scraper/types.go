package scraper

type Card struct {
	Title        string
	URL          string
	ThumbnailURL string
	Text         string
	DateRaw      string
	SequenceNum  int
}

type Selectors struct {
	ListContainer  string   `yaml:"list_container"`
	CardSelectors  string   `yaml:"card_selectors"`
	TitleSelectors []string `yaml:"title_selectors"`
	URLSelectors   []string `yaml:"url_selectors"`
	ImageSelectors []string `yaml:"image_selectors"`
	TextSelectors  []string `yaml:"text_selectors"`
	DateSelectors  []string `yaml:"date_selectors"`
	NextPageLink   []string `yaml:"next_page_link"`
}
