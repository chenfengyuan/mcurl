package http_util

import (
	// "crypto/md5"
	// "errors"
	"curl_cmd"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
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

var open_file_func openFileFunc = NewFile

const (
	BlockSize         int64 = 1024 * 1024
	NBlocksPerRequest       = 100
)

type DownloadRange struct {
	Start  int64
	Length int64
}

type File interface {
	Size() int64
	Name() string
	io.ReadWriteSeeker
	Truncate(size int64) error
	WriteAt([]byte, int64) (int, error)
	Sync() error
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
			out <- buf[:base]
			close(out)
			return
		} else {
			if downloaded >= length {
				out <- buf[:base-int(downloaded-length)]
				close(out)
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

func Downloader(in <-chan DownloadTaskInfo, out chan<- DownloadChunk, wg *sync.WaitGroup) {
	for task_info := range in {
		length := task_info.Length
		start := task_info.Start
		name := task_info.Name
		offset := task_info.RequestBaseN
		var downloaded int64 = 0
		for {
			chunk_datas := make(chan []byte, 1)
			go RangeGet(task_info.Requests[offset%len(task_info.Requests)], start, length, chunk_datas)
			for chunk_data := range chunk_datas {
				downloaded += int64(len(chunk_data))
				out <- DownloadChunk{Data: chunk_data, Name: name, Start: start}
				start += int64(len(chunk_data))
			}
			if downloaded == length {
				break
			} else {
				offset += 1
			}
		}
	}
	wg.Done()
}

func DownloadChunkWaitGroupAutoCloser(wg *sync.WaitGroup, c chan<- DownloadChunk) {
	wg.Wait()
	close(c)
}

// func Dispatcher(curl_cmds []string, n_workers int) {
// }

func Receiver(fileDownloadInfoC <-chan FileDownloadInfo, chunks <-chan DownloadChunk, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	fileDownloadInfoMap := make(map[string]FileDownloadInfo)
	fileFdMap := make(map[string]File)
	for {
		select {
		case info, ok := <-fileDownloadInfoC:
			if !ok {
				fileDownloadInfoC = nil
				continue
			}
			fileDownloadInfoMap[info.Name] = info
			fd, err := open_file_func(info.Name)
			if err != nil {
				log.Fatalf("can't open %v", info.Name)
			}
			fileFdMap[info.Name] = fd
		case chunk, ok := <-chunks:
			if !ok {
				return
			}
			info, ok := fileDownloadInfoMap[chunk.Name]
			info.Update(chunk.Start, int64(len(chunk.Data)))
			info.Sync()
			if !ok {
				log.Fatalf("can't find chunk.Name in info_map, %v", chunk.Name)
			}
			fd, ok := fileFdMap[chunk.Name]
			if !ok {
				log.Fatalf("can't find chunk.Name in file_fd_map, %v", chunk.Name)
			}
			_, err := fd.WriteAt(chunk.Data, chunk.Start)
			if err != nil {
				log.Fatalf("can't write content to file : %v", chunk.Name)
			}
			err = fd.Sync()
			if err != nil {
				log.Fatalf("can't write content to file : %v", chunk.Name)
			}
		}
	}
	log.Print("done")
}
func Run(curl_cmd_strs []string) {
	file_download_info_c := make(chan FileDownloadInfo, 1)
	task_info_c := make(chan DownloadTaskInfo, 1)
	chunk_c := make(chan DownloadChunk, 1)
	receiver_wait_group := sync.WaitGroup{}
	worker_wait_group := sync.WaitGroup{}
	receiver_wait_group.Add(1)
	go Receiver(file_download_info_c, chunk_c, &receiver_wait_group)
	worker_wait_group.Add(1)
	go Downloader(task_info_c, chunk_c, &worker_wait_group)
	worker_wait_group.Add(1)
	go Downloader(task_info_c, chunk_c, &worker_wait_group)
	go DownloadChunkWaitGroupAutoCloser(&worker_wait_group, chunk_c)

	for _, curl_cmd_str := range curl_cmd_strs {
		url := curl_cmd.ParseCmdStr(curl_cmd_str)[1]
		header := curl_cmd.GetHeadersFromCurlCmd(curl_cmd_str)
		req := Request{url, header}
		reqs := []Request{req}
		resource_info, err := GetResourceInfo(url, header)
		if err != nil {
			log.Fatal(err)
		}
		file_downloaded_info, err := NewFileDownloadInfo(resource_info.filename, resource_info.length)
		if err != nil {
			log.Fatal(err)
		}
		file_downloaded_info.Sync()
		file_download_info_c <- *file_downloaded_info
		for _, range_ := range file_downloaded_info.UndownloadedRanges() {
			task := DownloadTaskInfo{range_, reqs, resource_info.filename, 0}
			task_info_c <- task
		}
	}
	close(task_info_c)
	close(file_download_info_c)
	receiver_wait_group.Wait()
	worker_wait_group.Wait()
}
