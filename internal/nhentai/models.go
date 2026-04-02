package nhentai

type SearchResponse struct {
	Total    int             `json:"total"`
	PerPage  int             `json:"per_page"`
	NumPages int             `json:"num_pages"`
	Result   []GallerySearch `json:"result"`
}

type GallerySearch struct {
	ID       int64  `json:"id"`
	MediaID  string `json:"media_id"`
	NumPages int    `json:"num_pages"`
}

type Gallery struct {
	ID         int64        `json:"id"`
	MediaID    string       `json:"media_id"`
	Title      GalleryTitle `json:"title"`
	NumPages   int          `json:"num_pages"`
	UploadDate int64        `json:"upload_date"`
	Tags       []Tag        `json:"tags"`
	Pages      []Page       `json:"pages"`
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

type Page struct {
	Number int    `json:"number"`
	Path   string `json:"path"`
}
