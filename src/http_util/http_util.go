package http_util

import (
	// "crypto/md5"
	// "errors"
	"fmt"
	// "io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
func get(url_ string, header http.Header) (resp *http.Response, err error) {
	client := http.Client{CheckRedirect: CheckRedirect}
	req, err := http.NewRequest("GET", url_, nil)
	if err != nil {
		return
	}
	for i := 0; i < 10; i += 1 {
		log.Printf("url: %v\n\n", req.URL)
		req.Header = header
		resp, err = client.Do(req)
		if err != nil {
			switch err.(*url.Error).Err.(type) {
			case RedirectError:
				req.URL, err = resp.Location()
				if err != nil {
					return
				}
				continue
			default:
				return
			}
		} else {
			return
		}
	}
	return
}
func get_content_length(header http.Header) (rv int64, err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("can't get content length, Content-Length : %v", header["Content-Length"])
		}
	}()
	rv, err = strconv.ParseInt(header["Content-Length"][0], 10, 64)
	return rv, err
}
func get_attchment_filename(header http.Header) (fn string, err error) {
	content_disposition := header["Content-Disposition"]
	if len(content_disposition) != 1 {
		fmt.Printf("wrong Content-Disposition :%v", content_disposition)
		return
	}
	re := regexp.MustCompile(`filename="(.+)"`)
	tmp := re.FindAllStringSubmatch(content_disposition[0], 1)
	if len(tmp) != 1 && len(tmp[0]) != 2 {
		fmt.Printf("wrong Content-Disposition :%v", content_disposition)
		return
	}
	fn = tmp[0][1]
	re = regexp.MustCompile("/|\x00")
	fn = re.ReplaceAllString(fn, " ")
	return
}
func GetResourceStat(url_ string, header http.Header) (resource_stat ResourceStat, err error) {
	resp, err := get(url_, header)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	resource_stat.length, err = get_content_length(resp.Header)
	if err != nil {
		return
	}
	resource_stat.filename, _ = get_attchment_filename(resp.Header)
	return
}
