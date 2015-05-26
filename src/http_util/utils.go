package http_util

import (
	"fmt"
	// "log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var Get func(string, http.Header) (*http.Response, error) = get

func get(url_ string, header http.Header) (*http.Response, error) {
	for i := 0; i < 10; i += 1 {
		client := http.Client{CheckRedirect: CheckRedirect}
		req, err := http.NewRequest("GET", url_, nil)
		if err != nil {
			return nil, err
		}
		// log.Printf("url: %v\n\n", req.URL)
		req.Header = header
		resp, err := client.Do(req)
		// log.Print(resp.Header)
		if err != nil {
			switch err.(*url.Error).Err.(type) {
			case RedirectError:
				tmp, err := resp.Location()
				if err != nil {
					return nil, err
				}
				url_ = tmp.String()
				continue
			default:
				return nil, err
			}
		} else {
			return resp, err
		}
	}
	return nil, fmt.Errorf("unknow error")
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
		return "", fmt.Errorf("wrong Content-Disposition :%v", content_disposition)
	}
	re := regexp.MustCompile(`filename=?(.+)?`)
	tmp := re.FindAllStringSubmatch(content_disposition[0], 1)
	if len(tmp) != 1 || len(tmp[0]) != 2 {
		return "", fmt.Errorf("wrong Content-Disposition :%v", content_disposition)
	}
	fn = tmp[0][1]
	re = regexp.MustCompile("/|\x00|\"|'")
	fn = re.ReplaceAllString(fn, " ")
	return
}
func GetResourceInfo(url_ string, header http.Header) (resource_info ResourceInfo, err error) {
	resp, err := get(url_, header)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	resource_info.length, err = get_content_length(resp.Header)
	if err != nil {
		return
	}
	resource_info.filename, err = get_attchment_filename(resp.Header)
	if err != nil {
		return
	}
	return
}

func ConvertWaitGroupToBoolChan(wg *sync.WaitGroup, c chan<- bool) {
	wg.Wait()
	c <- true
}

func GetNowEpochInMilli() int64 {
	now := time.Now()
	return now.UnixNano() / 1000000
}
