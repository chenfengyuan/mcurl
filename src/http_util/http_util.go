package http_util

import (
	// "crypto/md5"
	// "errors"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	// "log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
)

const ChromeUserAgent string = "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.17 (KHTML, like Gecko) Chrome/24.0.1312.56 Safari/537.17"

type ResourceInfo struct {
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

var Get func(string, http.Header) (*http.Response, error) = get

func get(url_ string, header http.Header) (resp *http.Response, err error) {
	client := http.Client{CheckRedirect: CheckRedirect}
	req, err := http.NewRequest("GET", url_, nil)
	if err != nil {
		return
	}
	for i := 0; i < 10; i += 1 {
		// log.Printf("url: %v\n\n", req.URL)
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
	resource_info.filename, _ = get_attchment_filename(resp.Header)
	return
}

type FileDownloadInfo struct {
	Length int64
	MD5    string
	Blocks []bool
	Name   string
}

var open_file_func openFileFunc = NewFile

const (
	BlockSize         int64 = 1024 * 1024
	NBlocksPerRequest       = 100
)

func (info *FileDownloadInfo) Sync() error {
	data, err := json.Marshal(*info)
	if err != nil {
		return err
	}
	f, err := open_file_func(info.Name + ".info")
	if err != nil {
		return err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	err = f.Truncate(0)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	Truncate(info.Name, info.Length)
	return nil
}

func (info *FileDownloadInfo) Update(start int64, length int64) {
	end := start + length
	for i := start / BlockSize; i < int64(len(info.Blocks)); i++ {
		tmp := (i+1)*BlockSize - 1
		if tmp >= start && tmp < end {
			info.Blocks[i] = true
		}
	}
	if info.Length <= end {
		info.Blocks[len(info.Blocks)-1] = true
	}
}

type DownloadRange struct {
	Start  int64
	Length int64
}

func (info *FileDownloadInfo) UndownloadedRanges() []DownloadRange {
	rv := make([]DownloadRange, 0)
	i := 0
	for i < len(info.Blocks) {
		if info.Blocks[i] == true {
			i++
			continue
		}
		j := i
		for ; j < len(info.Blocks) && info.Blocks[j] == false; j++ {
			if j-i >= NBlocksPerRequest {
				break
			}
		}
		if j == len(info.Blocks) {
			rv = append(rv, DownloadRange{int64(i) * int64(BlockSize), int64(info.Length) - int64(i)*BlockSize})
		} else {
			rv = append(rv, DownloadRange{int64(i) * int64(BlockSize), int64(j-i) * int64(BlockSize)})
		}
		i = j
	}
	return rv
}

type File interface {
	Size() int64
	Name() string
	io.ReadWriteSeeker
	Truncate(size int64) error
}

type openFileFunc func(string) (File, error)

type FileS struct {
	os.File
}

func (f *FileS) Size() int64 {
	stat, err := f.Stat()
	if err != nil {
		return 0
	}
	return stat.Size()
}

func NewFile(name string) (File, error) {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0666)
	fs := FileS{*f}
	return &fs, err
}

func Truncate(name string, size int64) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	f.Close()
	err = os.Truncate(name, size)
	return err
}

func NewFileDownloadInfo(name string, file_size int64) (*FileDownloadInfo, error) {
	info_file, err := open_file_func(name + ".info")
	if err != nil {
		return nil, err
	}
	if info_file.Size() > 0 {
		info := FileDownloadInfo{}
		data, err := ioutil.ReadAll(info_file)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, &info)
		if err != nil {
			return nil, err
		}
		return &info, nil
	}
	tmp := new(FileDownloadInfo)
	tmp.Name = name
	tmp.Length = file_size
	n_blocks := tmp.Length / BlockSize
	if tmp.Length%BlockSize != 0 {
		n_blocks += 1
	}
	tmp.Blocks = make([]bool, n_blocks)
	file, err := open_file_func(name)
	if err != nil {
		return nil, err
	}
	tmp.Update(0, file.Size())
	return tmp, nil
}

type Request struct {
	url    string
	header http.Header
}

type DownloadTaskInfo struct {
	DownloadRange
	Requests     []Request
	Name         string
	RequestBaseN int
}

type DownloadChunk struct {
	Data  []byte
	Start int64
	Name  string
}

func RangeGet(req Request, start, length int64, out chan<- []byte) {
	header := http.Header{}
	for k, vs := range req.header {
		for _, v := range vs {
			header.Add(k, v)
		}
	}
	header.Add("Range", fmt.Sprintf("bytes=%d-", start))
	resp, err := get(req.url, header)
	if err != nil {
		out <- nil
		return
	}
	if resp.StatusCode != 206 {
		out <- nil
		return
	}
	buf := make([]byte, BlockSize)
	var downloaded int64 = 0
	base := 0
	for {
		n, err := resp.Body.Read(buf[base:])
		downloaded += int64(n)
		base += n
		if err != nil {
			out <- buf[:base]
			out <- nil
			return
		} else {
			if downloaded >= length {
				out <- buf[:base-int(downloaded-length)]
				out <- nil
				return
			}
			if int64(base) >= BlockSize {
				out <- buf[:base]
				buf = make([]byte, BlockSize)
				base = 0
			}
		}
	}
}

func Downloader(in <-chan DownloadTaskInfo, out chan<- DownloadChunk) {
	task_info := <-in
	length := task_info.Length
	start := task_info.Start
	name := task_info.Name
	offset := task_info.RequestBaseN
	var downloaded int64 = 0
SourceSwitchLoop:
	for {
		chunk_datas := make(chan []byte, 1)
		go RangeGet(task_info.Requests[offset%len(task_info.Requests)], start, length, chunk_datas)
		for chunk_data := range chunk_datas {
			if chunk_data == nil {
				if downloaded == length {
					out <- DownloadChunk{Data: nil, Name: name, Start: start}
					return
				} else {
					offset += 1
					continue SourceSwitchLoop
				}
			} else {
				downloaded += int64(len(chunk_data))
				out <- DownloadChunk{Data: chunk_data, Name: name, Start: start}
				start += int64(len(chunk_data))
			}
		}
	}
}
