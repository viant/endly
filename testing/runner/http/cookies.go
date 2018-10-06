package http

import "net/http"

//Cookies represents cookie
type Cookies []*http.Cookie

//SetCookies sets cookie header
func SetCookies(source Cookies, target http.Header) {
	if len(source) == 0 {
		return
	}
	for _, cookie := range source {
		if v := cookie.String(); v != "" {
			target.Add("Cookie", v)
		}
	}
}

//IndexByName index cookie by name
func (c *Cookies) IndexByName() map[string]*http.Cookie {
	var result = make(map[string]*http.Cookie)
	for _, cookie := range *c {
		result[cookie.Name] = cookie
	}
	return result
}

//IndexByPosition index cookie by position
func (c *Cookies) IndexByPosition() map[string]int {
	var result = make(map[string]int)
	for i, cookie := range *c {
		result[cookie.Name] = i
	}
	return result
}

//AddCookies adds cookies
func (c *Cookies) AddCookies(cookies ...*http.Cookie) {
	if len(cookies) == 0 {
		return
	}
	var indexed = c.IndexByPosition()
	for _, cookie := range cookies {

		if position, has := indexed[cookie.Name]; has {
			(*c)[position] = cookie
		} else {
			*c = append(*c, cookie)
		}
	}
}

