package http_util

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var Get func(string, http.Header, time.Duration) (*http.Response, error) = get

func get(url_ string, header http.Header, timeout time.Duration) (*http.Response, error) {
	for i := 0; i < 10; i += 1 {
		client := http.Client{CheckRedirect: CheckRedirect}
		client.Timeout = timeout
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

func RangeGet(req Request, start, length int64, out chan<- []byte) {
	header := http.Header{}
	for k, vs := range req.header {
		for _, v := range vs {
			header.Add(k, v)
		}
	}
	last_downloaded_block_time := GetNowEpochInSecond()
	header.Add("Range", fmt.Sprintf("bytes=%d-", start))
	resp, err := get(req.url, header, 0)
	mutex := sync.Mutex{}
	finished := false
	defer func() {
		mutex.Lock()
		finished = true
		mutex.Unlock()
	}()
	go func() {
		for {
			time.Sleep(TimeoutOfPerBlockDownload)
			mutex.Lock()
			if finished {
				mutex.Unlock()
				return
			}
			now := GetNowEpochInSecond()
			if time.Second*time.Duration(now-last_downloaded_block_time) > TimeoutOfPerBlockDownload {
				log.Print("timeout")
				resp.Body.Close()
				break
			}
			mutex.Unlock()
		}
	}()
	if err != nil {
		close(out)
		return
	}
	defer func() {
		resp.Body.Close()
	}()
	if resp.Header.Get("Content-Range") == "" {
		close(out)
		return
	}
	// if resp.StatusCode != 206 {
	// 	close(out)
	// 	return
	// }
	buf := make([]byte, BlockSize)
	var downloaded int64 = 0
	base := 0
	for {
		n, err := resp.Body.Read(buf[base:])
		downloaded += int64(n)
		base += n
		if err != nil {
			log.Print(err)
			out <- buf[:base]
			close(out)
			return
		} else {
			if downloaded >= length {
				out <- buf[:base-int(downloaded-length)]
				mutex.Lock()
				last_downloaded_block_time = GetNowEpochInSecond()
				mutex.Unlock()
				close(out)
				return
			}
			if int64(base) >= BlockSize {
				out <- buf[:base]
				mutex.Lock()
				last_downloaded_block_time = GetNowEpochInSecond()
				mutex.Unlock()
				buf = make([]byte, BlockSize)
				base = 0
			}
		}
	}
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
	fn, err = url.QueryUnescape(fn)
	if err != nil {
		return
	}
	re = regexp.MustCompile("/|\x00")
	fn = re.ReplaceAllString(fn, " ")
	re = regexp.MustCompile("\"|'")
	fn = re.ReplaceAllString(fn, "")
	return
}
func GetResourceInfo(url_ string, header http.Header) (resource_info ResourceInfo, err error) {
	for i := 0; i < 3; i++ {
		var resp *http.Response
		resp, err = get(url_, header, TimeoutOfGetResourceInfo)
		if err != nil {
			if i < 2 {
				time.Sleep(time.Duration(10+30*i) * time.Second)
				continue
			}
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
func GetNowEpochInSecond() int64 {
	now := time.Now()
	return now.UnixNano() / 1000000000
}
