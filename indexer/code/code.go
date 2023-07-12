package code

type CodeFetcher interface {
	Get(contractAddress string) (code string)
	GetAfter(timestamp int64) (codeByAddress map[string]string)
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

func (f *CodeFetcherUrl) Get(contractAddress string) (code string) {
	if len(f.url) == 0 {
		return
	}
	if f.cache[contractAddress] != "" {
		return f.cache[contractAddress]
	}
	// TODO: fetch code from url

	return ""
}

func (f *CodeFetcherUrl) GetAfter(timestamp int64) (codeByAddress map[string]string) {

	return
}
