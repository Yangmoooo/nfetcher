package nhentai

type SearchResponse struct {
	Total    int             `json:"total"`
	PerPage  int             `json:"per_page"`
	NumPages int             `json:"num_pages"`
	Result   []GallerySearch `json:"result"`
}

type DownloadResponse struct {
	URL       string `json:"url"`
	ExpiresAt int64  `json:"expires_at"`
}

func (r *SearchResponse) Normalize() {
	for index := range r.Result {
		r.Result[index].Normalize()
	}
}

type GallerySearch struct {
	ID            int64        `json:"id"`
	MediaID       string       `json:"media_id"`
	EnglishTitle  string       `json:"english_title"`
	JapaneseTitle *string      `json:"japanese_title"`
	Title         GalleryTitle `json:"-"`
	NumPages      int          `json:"num_pages"`
}

func (g *GallerySearch) Normalize() {
	if g.Title.English == "" {
		g.Title.English = g.EnglishTitle
	}
	if g.Title.Japanese == "" && g.JapaneseTitle != nil {
		g.Title.Japanese = *g.JapaneseTitle
	}
}

type Gallery struct {
	ID         int64        `json:"id"`
	MediaID    string       `json:"media_id"`
	Title      GalleryTitle `json:"title"`
	NumPages   int          `json:"num_pages"`
	UploadDate int64        `json:"upload_date"`
	Tags       []Tag        `json:"tags"`
}

type GalleryTitle struct {
	English  string `json:"english"`
	Japanese string `json:"japanese"`
}

type Tag struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}
