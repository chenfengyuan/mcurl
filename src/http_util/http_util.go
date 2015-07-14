package http_util

import (
	// "crypto/md5"
	// "errors"
	"bytes"
	"curl_cmd"
	"fmt"
	"http_info"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
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
	BlockSize                 int64 = 1024 * 1024
	TimeoutOfGetResourceInfo        = 60 * time.Second
	TimeoutOfPerBlockDownload       = 1024 / 42 * time.Second
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
	LastWorkerN int
}

type DownloadChunk struct {
	Data  []byte
	Start int64
	Name  string
}

func fileDownloader(file_info FileDownloadInfo, reqs []Request, worker_in <-chan int, worker_out chan<- int, chunk_c chan<- DownloadChunk, init_worker_n int) {
	ranges := file_info.UndownloadedRanges()
	worker_task_channel_map := make(map[int]chan DownloadTaskInfo)
	for i := 0; i < len(reqs); i++ {
		worker_task_channel_map[i] = make(chan DownloadTaskInfo)
	}
	failed_task_info_c := make(chan DownloadTaskInfo)
	task_finished_notification := make(chan int)
	num_of_worker_needed := init_worker_n
	if num_of_worker_needed > len(reqs) {
		num_of_worker_needed = len(reqs)
	}
	if num_of_worker_needed > len(ranges) {
		num_of_worker_needed = len(ranges)
	}
	if num_of_worker_needed == 0 {
		worker_out <- init_worker_n
		for x := range worker_in {
			worker_out <- x
		}
		close(worker_out)
		return
	}
	for i := 0; i < num_of_worker_needed && i < init_worker_n; i++ {
		task_info := DownloadTaskInfo{DownloadRange: ranges[i], Request: reqs[i], LastWorkerN: i, Name: file_info.Name}
		c := worker_task_channel_map[i]
		go downloader(c, chunk_c, task_finished_notification, failed_task_info_c)
		c <- task_info
	}
	max_worker_n := 0
	num_of_working_downloaders := 0
	if init_worker_n > num_of_worker_needed {
		worker_out <- init_worker_n - num_of_worker_needed
		max_worker_n = num_of_worker_needed - 1
		num_of_working_downloaders = max_worker_n + 1
		ranges = ranges[num_of_worker_needed:]
	} else {
		max_worker_n = init_worker_n - 1
		num_of_working_downloaders = max_worker_n + 1
		ranges = ranges[init_worker_n:]
	}
ForLoop:
	for {
		select {
		case new_available_worker_n, ok := <-worker_in:
			if !ok {
				worker_in = nil
			}
			for i := 0; i < new_available_worker_n; i++ {
				if max_worker_n >= num_of_worker_needed-1 {
					worker_out <- 1
				} else {
					if len(ranges) > 0 {
						num_of_working_downloaders++
						max_worker_n++
						task_info := DownloadTaskInfo{DownloadRange: ranges[0], Request: reqs[max_worker_n], LastWorkerN: max_worker_n, Name: file_info.Name}
						c := worker_task_channel_map[max_worker_n]
						go downloader(c, chunk_c, task_finished_notification, failed_task_info_c)
						c <- task_info
						ranges = ranges[1:]
					} else {
						worker_out <- 1
					}
				}
			}
		case wait_for_task_worker_n := <-task_finished_notification:
			if len(ranges) > 0 {
				task_info := DownloadTaskInfo{DownloadRange: ranges[0], Request: reqs[wait_for_task_worker_n], LastWorkerN: wait_for_task_worker_n, Name: file_info.Name}
				c := worker_task_channel_map[wait_for_task_worker_n]
				go downloader(c, chunk_c, task_finished_notification, failed_task_info_c)
				c <- task_info
				ranges = ranges[1:]
			} else {
				num_of_working_downloaders--
				worker_out <- 1
				close(worker_task_channel_map[wait_for_task_worker_n])
				if num_of_working_downloaders == 0 {
					break ForLoop
				}
			}
		case failed_task_info := <-failed_task_info_c:
			worker_out <- 1
			num_of_working_downloaders--
			close(worker_task_channel_map[failed_task_info.LastWorkerN])
			if num_of_working_downloaders > 0 {
				ranges = append(ranges, failed_task_info.DownloadRange)
			} else {
				break ForLoop
			}
		}
	}
	if worker_in != nil {
		for x := range worker_in {
			worker_out <- x
		}
	}
	close(worker_out)
}

func downloader(task_info_c <-chan DownloadTaskInfo, chunk_c chan<- DownloadChunk, task_finished_notification chan<- int, failed_task_info_c chan<- DownloadTaskInfo) {
	for task_info := range task_info_c {
		worker_n := task_info.LastWorkerN
		length := task_info.Length
		start := task_info.Start
		name := task_info.Name
		url_obj, err := url.Parse(task_info.Request.url)
		if err != nil {
			log.Fatalf("Failed to parse url %v, %v", task_info.Request.url, err)
		}
		host := url_obj.Host
		log.Printf("D[%v] %v %v %v", name, host, worker_n, task_info.DownloadRange)
		var downloaded int64 = 0
		task_start_time := GetNowEpochInMilli()
		for try_times := 0; try_times < 100; try_times++ {
			if try_times > 0 {
				time.Sleep(time.Second * time.Duration(60))
			}
			if try_times > 10 {
				time.Sleep(time.Second * time.Duration(60*2))
			}
			chunk_datas := make(chan []byte)
			go RangeGet(task_info.Request, start, length-(start-task_info.Start), chunk_datas)
			for chunk_data := range chunk_datas {
				downloaded += int64(len(chunk_data))
				log.Printf("D[%v] %v %v %v k/s %v%%", name, host, worker_n, downloaded*1000/1024/(GetNowEpochInMilli()-task_start_time), 100*downloaded/length)
				chunk_c <- DownloadChunk{Data: chunk_data, Name: name, Start: start}
				start += int64(len(chunk_data))
			}
			if downloaded == length {
				task_finished_notification <- worker_n
				break
			}
		}
		if downloaded < length {
			task_info.Start = start
			task_info.Length = length - downloaded
			failed_task_info_c <- task_info
		}
	}
}

func Receiver(fileDownloadInfoC <-chan FileDownloadInfo, chunks <-chan DownloadChunk, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	info := ""
	info_mutex := sync.Mutex{}
	go http_info.Server(&info, &info_mutex)
	fileDownloadInfoMap := make(map[string]FileDownloadInfo)
	fileFdMap := make(map[string]File)
	update_info := func() {
		buf := bytes.NewBuffer(nil)
		for filename, dinfo := range fileDownloadInfoMap {
			a := len(dinfo.Blocks)
			finished := 0
			for _, x := range dinfo.Blocks {
				if x == true {
					finished++
				}
			}
			fmt.Fprintf(buf, "%v %v%% %vMB\n", filename, 100*finished/a, finished)
		}
		info = buf.String()
	}
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
			info.Update(chunk.Start, int64(len(chunk.Data)))
			err = info.Sync()
			if err != nil {
				log.Fatalf("can't sync info, %v", err)
			}
			if info.Finished() {
				log.Printf("finished")
				delete(fileDownloadInfoMap, chunk.Name)
				fd.Close()
				delete(fileFdMap, chunk.Name)
				// log.Print(info, info.Finished(), len(fileDownloadInfoMap))
				if len(fileDownloadInfoMap) == 0 {
					return
				}
			}
			update_info()
		}
	}
}

func Run(curl_cmd_strs []string, num_of_workers int) {
	chunk_c := make(chan DownloadChunk, 1)
	file_name_reqs_map := make(map[string]*[]Request)
	file_download_info_c := make(chan FileDownloadInfo, 1)
	file_download_infos := []FileDownloadInfo{}
	receiver_wait_group := sync.WaitGroup{}

	receiver_wait_group.Add(1)
	go Receiver(file_download_info_c, chunk_c, &receiver_wait_group)

	for _, curl_cmd_str := range curl_cmd_strs {
		url := curl_cmd.ParseCmdStr(curl_cmd_str)[1]
		// log.Printf("%v %v", url, worker_n)
		header := curl_cmd.GetHeadersFromCurlCmd(curl_cmd_str)
		req := Request{url, header}
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
	}
	if len(file_download_infos) == 0 {
		return
	}
	for file_name, reqs := range file_name_reqs_map {
		log.Printf("%v %v", file_name, len(*reqs))
	}
	worker_in := make(chan int)
	worker_out := make(chan int)
	go fileDownloader(file_download_infos[0], *file_name_reqs_map[file_download_infos[0].Name], worker_in, worker_out, chunk_c, num_of_workers)
	file_download_infos = file_download_infos[1:]
	close(worker_in)
	for _, file_download_info := range file_download_infos {
		availabe_worker_n := <-worker_out
		new_worker_out := make(chan int)
		go fileDownloader(file_download_info, *file_name_reqs_map[file_download_info.Name], worker_out, new_worker_out, chunk_c, availabe_worker_n)
		worker_out = new_worker_out
	}
	for x := range worker_out {
		log.Print(x)
	}
	close(chunk_c)
	receiver_wait_group.Wait()
	// 	all_task_finish_c := make(chan bool)
	// 	go ConvertWaitGroupToBoolChan(&task_info_wait_group, all_task_finish_c)
	// 	close(file_download_info_c)
	// 	for file_name, reqs := range file_name_reqs_map {
	// 		log.Printf("%v %v", file_name, len(*reqs))
	// 	}
	// 	worker_n = -1
	// 	for _, task_info := range task_infos {
	// 		worker_n++
	// 		reqs := file_name_reqs_map[task_info.Name]
	// 		req := (*reqs)[worker_n%len(*reqs)]
	// 		task_info.Request = req
	// 		// log.Printf("%v %v %v", task_info.Name, task_info.DownloadRange, url_chan_map[task_info.url])
	// 		select {
	// 		case downloader_task_info_cs[url_chan_map[task_info.url]] <- task_info:
	// 		case failed_task_info := <-failed_task_info_c:
	// 			if failed_task_info.FailedTimes < 3 {
	// 				reqs := file_name_reqs_map[failed_task_info.Name]
	// 				last_worker_n := failed_task_info.LastWorkerN
	// 				new_req := (*reqs)[(last_worker_n+1)%len(*reqs)]
	// 				failed_task_info.Request = new_req
	// 				downloader_task_info_cs[url_chan_map[new_req.url]] <- failed_task_info
	// 			} else {
	// 				task_info_wait_group.Done()
	// 			}
	// 		}
	// 	}
	// ForLoop:
	// 	for {
	// 		select {
	// 		case failed_task_info := <-failed_task_info_c:
	// 			if failed_task_info.FailedTimes < 3 {
	// 				reqs := file_name_reqs_map[failed_task_info.Name]
	// 				last_worker_n := failed_task_info.LastWorkerN
	// 				new_req := (*reqs)[(last_worker_n+1)%len(*reqs)]
	// 				failed_task_info.Request = new_req
	// 				downloader_task_info_cs[url_chan_map[new_req.url]] <- failed_task_info
	// 			} else {
	// 				task_info_wait_group.Done()
	// 			}
	// 		case <-all_task_finish_c:
	// 			for _, chan_ := range downloader_task_info_cs {
	// 				close(chan_)
	// 			}
	// 			close(chunk_c)
	// 			close(failed_task_info_c)
	// 			break ForLoop
	// 		}
	// 	}
}
