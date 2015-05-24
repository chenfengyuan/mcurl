package http_util

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type ResourceStat struct {
	length   int64
	filename string
}
type RedirectError struct {
	s string
}

func (re RedirectError) Error() string {
	return re.s
}

func CheckRedirect(*http.Request, []*http.Request) error {
	return RedirectError{"don't follow redirect"}
}

type myUrlEror url.Error

func (e myUrlEror) Error() string {
	return (&e).Error()
}
func GetResourceStat(url_ string, header http.Header) (resource_stat ResourceStat, rv error) {
	rv = errors.New("unknow")
	client := http.Client{CheckRedirect: CheckRedirect}
	req, err := http.NewRequest("GET", url_, nil)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	var resp *http.Response
	for i := 0; i < 5; i += 1 {
		fmt.Printf("url: %v\n\n", req.URL)
		req.Header = header
		resp, err = client.Do(req)
		if err != nil {
			switch err.(*url.Error).Err.(type) {
			case RedirectError:
				fmt.Printf("new url")
				req.URL, err = resp.Location()
				if err != nil {
					fmt.Printf("error: %v", err)
					return
				}
				continue
			default:
				fmt.Printf("error: %v", err)
				return
			}
		}
		break
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	body_str := string(body)
	if err != nil {
		fmt.Printf("error : %v", err)
	} else {
		fmt.Printf("headers: %v\ndata : \n%v", resp.Header, body_str)
		h := md5.New()
		h.Write(body)
		fmt.Printf("%x", h.Sum(nil))
	}
	return
}
