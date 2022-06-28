package v1

type Info struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	CreationDate int64  `json:"creationDate"`
	Files        []File `json:"files"`
}

type File struct {
	Path   string `json:"path"`
	Length int64  `json:"length"`
}
