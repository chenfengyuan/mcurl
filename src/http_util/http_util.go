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
	Close() error
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
	Request
	Name        string
	FailedTimes int
	LastWorkerN int
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

func Downloader(task_info_c <-chan DownloadTaskInfo, finished_c chan<- DownloadChunk, failed_task_info_c chan<- DownloadTaskInfo, wg *sync.WaitGroup, worker_n int) {
	for task_info := range task_info_c {
		log.Printf("Worker %v, %v", worker_n, task_info)
		length := task_info.Length
		start := task_info.Start
		name := task_info.Name
		var downloaded int64 = 0
		for try_times := 0; try_times < 3; try_times++ {
			chunk_datas := make(chan []byte, 1)
			go RangeGet(task_info.Request, start, length, chunk_datas)
			for chunk_data := range chunk_datas {
				downloaded += int64(len(chunk_data))
				finished_c <- DownloadChunk{Data: chunk_data, Name: name, Start: start}
				start += int64(len(chunk_data))
			}
			if downloaded == length {
				break
			}
		}
		if downloaded < length {
			task_info.Start = start
			task_info.Length = length - downloaded
			task_info.LastWorkerN = worker_n
			task_info.FailedTimes++
			failed_task_info_c <- task_info
		}
	}
	wg.Done()
}

func DownloadChunkWaitGroupAutoCloser(wg *sync.WaitGroup, c chan<- DownloadChunk) {
	wg.Wait()
	close(c)
}

func Receiver(fileDownloadInfoC <-chan FileDownloadInfo, chunks <-chan DownloadChunk, finished chan<- int) {
	defer func() {
		finished <- 0
	}()
	fileDownloadInfoMap := make(map[string]FileDownloadInfo)
	fileFdMap := make(map[string]File)
	for {
		select {
		case info, ok := <-fileDownloadInfoC:
			if !ok {
				if len(fileDownloadInfoMap) == 0 {
					return
				}
				fileDownloadInfoC = nil
				break
			}
			if info.Finished() {
				break
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
			if info.Finished() {
				delete(fileDownloadInfoMap, chunk.Name)
				fd.Close()
				delete(fileFdMap, chunk.Name)
				log.Print(info, info.Finished(), len(fileDownloadInfoMap))
				if len(fileDownloadInfoMap) == 0 {
					return
				}
			}
		}
	}
}

func Run(curl_cmd_strs []string, num_of_workers int) {
	file_download_info_c := make(chan FileDownloadInfo, 1)
	downloader_task_info_cs := make([]chan DownloadTaskInfo, num_of_workers)
	chunk_c := make(chan DownloadChunk, 1)
	url_chan_map := make(map[string]int)
	file_name_reqs_map := make(map[string]*[]Request)
	receiver_finish_channel := make(chan int, 1)
	downloader_wait_group := sync.WaitGroup{}
	failed_task_info_c := make(chan DownloadTaskInfo, 1)

	downloader_wait_group.Add(num_of_workers)

	go Receiver(file_download_info_c, chunk_c, receiver_finish_channel)
	for i := 0; i < num_of_workers; i++ {
		tmp := make(chan DownloadTaskInfo, 1)
		downloader_task_info_cs[i] = tmp
		go Downloader(tmp, chunk_c, failed_task_info_c, &downloader_wait_group, i)
	}
	go DownloadChunkWaitGroupAutoCloser(&downloader_wait_group, chunk_c)

	file_download_infos := []FileDownloadInfo{}
	// resource_infos := []ResourceInfo{}
	worker_n := -1
	task_infos := []DownloadTaskInfo{}
	for _, curl_cmd_str := range curl_cmd_strs {
		worker_n++
		worker_n = worker_n % num_of_workers
		url := curl_cmd.ParseCmdStr(curl_cmd_str)[1]
		url_chan_map[url] = worker_n
		header := curl_cmd.GetHeadersFromCurlCmd(curl_cmd_str)
		req := Request{url, header}
		// reqs = append(reqs, req)
		// reqs := []Request{req}
		resource_info, err := GetResourceInfo(url, header)
		if err != nil {
			log.Printf("Can't get resource_info of url(%v), err(%v)", url, err)
			continue
		}
		file_name_reqs, ok := file_name_reqs_map[resource_info.filename]
		if !ok {
			tmp := make([]Request, 0, 1)
			file_name_reqs = &tmp
			file_name_reqs_map[resource_info.filename] = file_name_reqs
			*file_name_reqs = append(*file_name_reqs, req)
		} else {
			*file_name_reqs = append(*file_name_reqs, req)
			continue
		}

		log.Printf("Get Resource %v %v", resource_info.filename, resource_info.length)
		file_download_info, err := NewFileDownloadInfo(resource_info.filename, resource_info.length)
		if err != nil {
			log.Printf("Can't create file downloaded info of %v, %v", resource_info.filename, resource_info.length)
			continue
		}
		err = file_download_info.Sync()
		if err != nil {
			log.Printf("Can't sync file downloaded info of %v %v", resource_info.filename, resource_info.length)
			continue
		}
		file_download_info_c <- *file_download_info
		file_download_infos = append(file_download_infos, *file_download_info)
		for _, range_ := range file_download_info.UndownloadedRanges() {
			task := DownloadTaskInfo{range_, req, resource_info.filename, 0, worker_n}
			task_infos = append(task_infos, task)
		}
	}
	close(file_download_info_c)
	for file_name, reqs := range file_name_reqs_map {
		log.Printf("%v %v", file_name, len(*reqs))
	}
	for _, task_info := range task_infos {
		log.Printf("%v %v", task_info.Name, task_info.DownloadRange)
		select {
		case downloader_task_info_cs[url_chan_map[task_info.url]] <- task_info:
		case failed_task_info := <-failed_task_info_c:
			if failed_task_info.FailedTimes < 3 {
				reqs := file_name_reqs_map[failed_task_info.Name]
				last_worker_n := failed_task_info.LastWorkerN
				new_req := (*reqs)[(last_worker_n+1)%len(*reqs)]
				failed_task_info.Request = new_req
				downloader_task_info_cs[url_chan_map[new_req.url]] <- failed_task_info
			}
		}
	}
ForLoop:
	for {
		select {
		case failed_task_info := <-failed_task_info_c:
			if failed_task_info.FailedTimes < 3 {
				reqs := file_name_reqs_map[failed_task_info.Name]
				last_worker_n := failed_task_info.LastWorkerN
				new_req := (*reqs)[(last_worker_n+1)%len(*reqs)]
				failed_task_info.Request = new_req
				downloader_task_info_cs[url_chan_map[new_req.url]] <- failed_task_info
			}
		case <-receiver_finish_channel:
			for _, chan_ := range downloader_task_info_cs {
				close(chan_)
			}
			break ForLoop
		}
	}
	downloader_wait_group.Wait()
}
