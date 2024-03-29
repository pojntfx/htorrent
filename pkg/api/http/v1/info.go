package v1

type Info struct {
	Name         string `json:"name"`
	InfoHash     string `json:"infohash"`
	Description  string `json:"description"`
	CreationDate int64  `json:"creationDate"`
	Files        []File `json:"files"`
}

type File struct {
	Path   string `json:"path"`
	Length int64  `json:"length"`
}

type TorrentMetrics struct {
	Magnet   string        `json:"magnet"`
	InfoHash string        `json:"infohash"`
	Peers    int           `json:"peers"`
	Files    []FileMetrics `json:"files"`
}

type FileMetrics struct {
	Path      string `json:"path"`
	Length    int64  `json:"length"`
	Completed int64  `json:"completed"`
}
