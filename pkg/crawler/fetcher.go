package crawler

import (
	"io/ioutil"
	"net/http"
)

type Fetcher interface {
	Fetch(url string) (result Result, err error)
}

type Result struct {
	Url  string
	Body string
}

type DefaultFetcher struct{}

func Create() Fetcher {
	return &DefaultFetcher{}
}

func (f DefaultFetcher) Fetch(url string) (Result, error) {
	resp, err := http.Get(url)
	if err != nil {
		return Result{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Url:  url,
		Body: string(body),
	}, nil
}
