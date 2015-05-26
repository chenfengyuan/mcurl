package http_util

import (
	"log"
	"net/http"
	"os"
	"reflect"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func TestResourceStat(t *testing.T) {
	info, err := GetResourceInfo("http://7xislp.com1.z0.glb.clouddn.com/b.mp3", http.Header{})
	if err != nil {
		t.Errorf("%v", err)
	} else {
		t.Logf("%v", info)
	}
	if info.length != 42 {
		t.Errorf("wrong content length, expect 42, get %v", info.length)
	}
	if info.filename != "b.mp3" {
		t.Errorf("wrong filename, expect b.mp3, get %v", info.filename)
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
	info, err := NewFileDownloadInfo(test_file, file_size)
	if err != nil {
		t.Fatal(err)
	}
	info.Sync()
	if info.Length != file_size {
		t.Fatalf("wrong file size, expect %v, get %v", file_size, info.Length)
	}
	if info.BlockSize != BlockSize {
		t.Fatalf("wrong block size, expect %v, get %v", BlockSize, info.BlockSize)
	}
	if !reflect.DeepEqual(info.Blocks, make([]bool, n_blocks)) {
		t.Fatalf("wrong Blocks, should be all false")
	}
	if info.Name != test_file {
		t.Fatalf("wrong filename, expect %v, get %v", test_file, info.Name)
	}
	ranges := info.UndownloadedRanges()
	t.Log(ranges[0])
	info.Blocks[1] = true
	ranges = info.UndownloadedRanges()
	t.Log(ranges[0])
	info.Sync()
	info, err = NewFileDownloadInfo(test_file, file_size)
	if err != nil {
		t.Fatal(err)
	}
	if info.Blocks[1] != true {
		t.Fatalf("wrong blocks, first element is not true")
	}

	// t.Logf("%#v", NewFileDownloadInfo("test.mp3"))
}
