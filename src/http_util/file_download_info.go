package http_util

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const (
	FileDownloadInfoSuffix     string = ".json"
	FileDownloadInfoTempSuffix string = ".temp.json"
)

type FileDownloadInfo struct {
	Length int64
	MD5    string
	Blocks []bool
	Name   string
}

func (info *FileDownloadInfo) Sync() error {
	data, err := json.Marshal(*info)
	if err != nil {
		return err
	}
	f, err := open_file_func(info.Name + FileDownloadInfoTempSuffix)
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
	os.Rename(info.Name+FileDownloadInfoTempSuffix, info.Name+FileDownloadInfoSuffix)
	// Truncate(info.Name, info.Length)
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
func (info *FileDownloadInfo) Finished() bool {
	for _, x := range info.Blocks {
		if x == false {
			return false
		}
	}
	return true
}

func NewFileDownloadInfo(name string, file_size int64) (*FileDownloadInfo, error) {
	info_file, err := open_file_func(name + FileDownloadInfoSuffix)
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
