package client

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	jsoniter "github.com/json-iterator/go"
	v1 "github.com/pojntfx/htorrent/pkg/api/http/v1"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type Manager struct {
	url      string
	username string
	password string
	ctx      context.Context
}

func NewManager(
	url string,
	username string,
	password string,
	ctx context.Context,
) *Manager {
	return &Manager{
		url:      url,
		username: username,
		password: password,
		ctx:      ctx,
	}
}

func (m *Manager) GetInfo(magnetLink string) ([]v1.File, error) {
	hc := &http.Client{}

	baseURL, err := url.Parse(m.url)
	if err != nil {
		return nil, err
	}

	infoSuffix, err := url.Parse("/info")
	if err != nil {
		return nil, err
	}

	info := baseURL.ResolveReference(infoSuffix)

	q := info.Query()
	q.Set("magnet", magnetLink)
	info.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, info.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(m.username, m.password)

	res, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.New(res.Status)
	}

	files := []v1.File{}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&files); err != nil {
		return nil, err
	}

	return files, nil
}
