package http_util

import (
	// "crypto/md5"
	// "errors"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
)

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
	Length    int64
	MD5       string
	BlockSize int64
	Blocks    []bool
	Name      string
}

var open_file_func openFileFunc = NewFile

const (
	BlockSize   = 1024 * 1024
	RequestSize = 1024 * 1024 * 100
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

type DownloadRange struct {
	Start  int64
	Length int64
}

func (info *FileDownloadInfo) UndownloadedRanges() []DownloadRange {
	rv := make([]DownloadRange, 0)
	i := 0
	for ; i < len(info.Blocks); i += 1 {
		if info.Blocks[i] == true {
			continue
		}
		j := i
		for ; j < len(info.Blocks) && info.Blocks[j] == false; j += 1 {
		}
		if j == len(info.Blocks) {
			rv = append(rv, DownloadRange{int64(i) * int64(BlockSize), int64(info.Length) - int64(i*BlockSize)})
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
	tmp.BlockSize = BlockSize
	n_blocks := tmp.Length / tmp.BlockSize
	if tmp.Length%tmp.BlockSize != 0 {
		n_blocks += 1
	}
	tmp.Blocks = make([]bool, n_blocks)
	return tmp, nil
}
