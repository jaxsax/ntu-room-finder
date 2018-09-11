package fetcher

import (
	"io/ioutil"
	"net/http"
)

type Fetcher interface {
	Fetch(url string) (result FetcherResult, err error)
}

type FetcherResult struct {
	Url  string
	Body string
}

type DefaultFetcher struct{}

func NewFetcher() *DefaultFetcher {
	return &DefaultFetcher{}
}

func (f *DefaultFetcher) Fetch(url string) (FetcherResult, error) {
	resp, err := http.Get(url)
	if err != nil {
		return FetcherResult{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return FetcherResult{}, err
	}

	return FetcherResult{
		Url:  url,
		Body: string(body),
	}, nil
}
