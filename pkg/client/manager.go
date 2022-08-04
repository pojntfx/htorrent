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

func (m *Manager) GetInfo(magnetLink string) (v1.Info, error) {
	hc := &http.Client{}

	baseURL, err := url.Parse(m.url)
	if err != nil {
		return v1.Info{}, err
	}

	infoSuffix, err := url.Parse("/info")
	if err != nil {
		return v1.Info{}, err
	}

	infoURL := baseURL.ResolveReference(infoSuffix)

	q := infoURL.Query()
	q.Set("magnet", magnetLink)
	infoURL.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, infoURL.String(), http.NoBody)
	if err != nil {
		return v1.Info{}, err
	}
	req.SetBasicAuth(m.username, m.password)

	res, err := hc.Do(req)
	if err != nil {
		return v1.Info{}, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	if res.StatusCode != http.StatusOK {
		return v1.Info{}, errors.New(res.Status)
	}

	info := v1.Info{}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&info); err != nil {
		return v1.Info{}, err
	}

	return info, nil
}

func (m *Manager) GetMetrics() ([]v1.TorrentMetrics, error) {
	hc := &http.Client{}

	baseURL, err := url.Parse(m.url)
	if err != nil {
		return []v1.TorrentMetrics{}, err
	}

	infoSuffix, err := url.Parse("/metrics")
	if err != nil {
		return []v1.TorrentMetrics{}, err
	}

	infoURL := baseURL.ResolveReference(infoSuffix)

	req, err := http.NewRequest(http.MethodGet, infoURL.String(), http.NoBody)
	if err != nil {
		return []v1.TorrentMetrics{}, err
	}
	req.SetBasicAuth(m.username, m.password)

	res, err := hc.Do(req)
	if err != nil {
		return []v1.TorrentMetrics{}, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	if res.StatusCode != http.StatusOK {
		return []v1.TorrentMetrics{}, errors.New(res.Status)
	}

	metrics := []v1.TorrentMetrics{}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&metrics); err != nil {
		return []v1.TorrentMetrics{}, err
	}

	return metrics, nil
}
