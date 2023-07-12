package code

type CodeFetcher interface {
	Search(contractAddress string) (code string)
	NewAfter(timestamp int64) (codeByAddress map[string]string)
}

func NewCodeFetcher(url string) CodeFetcher {
	return &CodeFetcherUrl{
		url:   url,
		cache: make(map[string]string),
	}
}

type CodeFetcherUrl struct {
	url        string
	cache      map[string]string
	lastUpdate uint64
}

func (f *CodeFetcherUrl) Search(contractAddress string) (code string) {
	if len(f.url) == 0 {
		return
	}
	if f.cache[contractAddress] != "" {
		return f.cache[contractAddress]
	}
	// TODO: fetch code from url

	return ""
}

func (f *CodeFetcherUrl) NewAfter(timestamp int64) (codeByAddress map[string]string) {

	return
}
