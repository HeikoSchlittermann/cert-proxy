package jar

import (
	"net/http"
	"net/url"
)

type Memory map[string][]*http.Cookie

func InMemory() Memory {
	return make(Memory, 1)
}

func (m Memory) Cookies(u *url.URL) []*http.Cookie {
	return m[u.Scheme+u.Host]
}

func (m Memory) SetCookies(u *url.URL, cookies []*http.Cookie) {
	m[u.Scheme+u.Host] = cookies
}
