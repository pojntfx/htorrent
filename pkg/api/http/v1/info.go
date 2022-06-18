package v1

type File struct {
	Path         string `json:"path"`
	Length       int64  `json:"length"`
	CreationDate int64  `json:"creationTime"`
}
