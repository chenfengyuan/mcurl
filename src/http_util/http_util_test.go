package http_util

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"sync"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func TestResourceStat(t *testing.T) {
	info, err := GetResourceInfo("http://7xislp.com1.z0.glb.clouddn.com/b.mp3", http.Header{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	if info.length != 42 {
		t.Fatalf("wrong content length, expect 42, get %v", info.length)
	}
	if info.filename != "b.mp3" {
		t.Fatalf("wrong filename, expect b.mp3, get %v", info.filename)
	}
}

func TestFileDownloadInfo(t *testing.T) {
	test_file := "test_file"
	var file_size int64 = 1024*1024*1024*3 + 1024*3 + 42
	n_blocks := 1024*3 + 1
	defer func() {
		os.Remove(test_file)
		os.Remove(test_file + ".info")
	}()
	Truncate(test_file, BlockSize)
	info, err := NewFileDownloadInfo(test_file, file_size)
	if err != nil {
		t.Fatal(err)
	}
	info.Sync()
	if info.Length != file_size {
		t.Fatalf("wrong file size, expect %v, get %v", file_size, info.Length)
	}
	if info.Blocks[0] != true {
		t.Fatalf("wrong Block, the first should be true")
	}
	info.Blocks[0] = false
	if !reflect.DeepEqual(info.Blocks, make([]bool, n_blocks)) {
		t.Fatalf("wrong Blocks, should be all false")
	}
	if info.Name != test_file {
		t.Fatalf("wrong filename, expect %v, get %v", test_file, info.Name)
	}
	ranges := info.UndownloadedRanges()
	if tmp_range := (DownloadRange{0, NBlocksPerRequest * BlockSize}); ranges[0] != tmp_range {
		t.Fatalf("wrong range, expect %v, get %v", tmp_range, ranges[0])
	}
	if tmp := 31; len(ranges) != tmp {
		t.Fatalf("wrong length of ranges, expect %v, get %v", tmp, len(ranges))
	}
	info.Update(BlockSize*2-1, 1)
	ranges = info.UndownloadedRanges()
	if tmp := 32; len(ranges) != tmp {
		t.Fatalf("wrong length of ranges, expect %v, get %v", tmp, len(ranges))
	}
	if tmp_range := (DownloadRange{0, BlockSize}); ranges[0] != tmp_range {
		t.Fatalf("wrong range, expect %v, get %v", tmp_range, ranges[0])
	}
	for i := 1; i < len(ranges)-1; i++ {
		if tmp_range := (DownloadRange{int64(i-1)*NBlocksPerRequest*BlockSize + BlockSize*2, NBlocksPerRequest * BlockSize}); tmp_range != ranges[i] {
			t.Fatalf("wrong range, expect %v, get #%v %v", tmp_range, i, ranges[i])
		}
	}
	if tmp_range := (DownloadRange{30*NBlocksPerRequest*BlockSize + BlockSize*2, info.Length - 30*NBlocksPerRequest*BlockSize - BlockSize*2}); tmp_range != ranges[31] {
		t.Fatalf("wrong range, expect %v, get #%v %v", tmp_range, 31, ranges[31])
	}
	info.Sync()
	info, err = NewFileDownloadInfo(test_file, file_size)
	if err != nil {
		t.Fatal(err)
	}
	if info.Blocks[1] != true {
		t.Fatalf("wrong blocks, first element is not true")
	}

	info.Update(file_size-1, 10)
	if x := info.Blocks[len(info.Blocks)-1]; x != true {
		t.Fatalf("wrong last blocks, should be true")
	}
}

func testDownloaderHelper(t *testing.T, info DownloadTaskInfo, md5sum string) {
	task_infos := make(chan DownloadTaskInfo, 1)
	task_infos <- info
	close(task_infos)
	chunks := make(chan DownloadChunk, 1)
	h := md5.New()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go DownloadChunkWaitGroupAutoCloser(&wg, chunks)
	go Downloader(task_infos, chunks, &wg)
	for chunk := range chunks {
		h.Write(chunk.Data)
	}
	if x := fmt.Sprintf("%x", h.Sum(nil)); x != md5sum {
		t.Fatalf("incorrect downloaded data, expect md5 %v,get %v", md5sum, x)
	}
}

func TestDownloader(t *testing.T) {
	header := http.Header{"User-Agent": []string{ChromeUserAgent}}
	req := Request{`http://dldir1.qq.com/qqfile/qq/QQ7.2/14810/QQ7.2.exe`, header}
	bad_req := Request{`http://127.0.0.1:10`, header}
	reqs := []Request{bad_req, req}
	info := DownloadTaskInfo{DownloadRange{0, BlockSize + 1}, reqs, "QQ7.2.exe", 0}
	testDownloaderHelper(t, info, "f3a593ffaecc91ee14f65a43692c12a5")

	info = DownloadTaskInfo{DownloadRange{1, BlockSize - 1}, reqs, "QQ7.2.exe", 0}
	testDownloaderHelper(t, info, "3bf86e395d79b8e5e06aa56adf6eb79d")

	info = DownloadTaskInfo{DownloadRange{1, BlockSize * 2}, reqs, "QQ7.2.exe", 0}
	testDownloaderHelper(t, info, "09273f055723fabae599736214886a06")

	info = DownloadTaskInfo{DownloadRange{1, BlockSize*3 - 1}, reqs, "QQ7.2.exe", 0}
	testDownloaderHelper(t, info, "7ebfa5deddab0c7b7f01f558695517f2")

	info = DownloadTaskInfo{DownloadRange{57179300, 20}, reqs, "QQ7.2.exe", 0}
	testDownloaderHelper(t, info, "4fb35a571508c9b8237bdbb71b4fb797")
}

func TestReceiver(t *testing.T) {
	header := http.Header{"User-Agent": []string{ChromeUserAgent}}
	req := Request{`http://dldir1.qq.com/qqfile/qq/QQ7.2/14810/QQ7.2.exe`, header}
	bad_req := Request{`http://127.0.0.1:10`, header}
	reqs := []Request{bad_req, req}
	var length int64 = 57179320
	file_name := "QQ7.2.exe"
	file_download_info, err := NewFileDownloadInfo(file_name, length)
	if err != nil {
		t.Fatal(err)
	}
	file_download_info.Sync()
	file_download_info_c := make(chan FileDownloadInfo, 1)
	file_download_info_c <- *file_download_info
	close(file_download_info_c)
	chunk_c := make(chan DownloadChunk, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go Receiver(file_download_info_c, chunk_c, &wg)
	task_info := DownloadTaskInfo{DownloadRange{57179300, 20}, reqs, "QQ7.2.exe", 0}
	task_info_c := make(chan DownloadTaskInfo, 1)
	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	go DownloadChunkWaitGroupAutoCloser(&wg2, chunk_c)
	go Downloader(task_info_c, chunk_c, &wg2)
	task_info_c <- task_info
	close(task_info_c)
	wg.Wait()
	if file_download_info.Blocks[len(file_download_info.Blocks)-1] != true {
		t.Fatalf("the last block should be true")
	}
	defer func() {
		test_file := "QQ7.2.exe"
		os.Remove(test_file)
		os.Remove(test_file + ".info")
	}()
}
