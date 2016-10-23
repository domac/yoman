package yoman

import (
	"errors"
	"os"
	"strings"
)

func CheckDataFileExist(filePath string) error {

	if filePath == "" {
		return errors.New("数据文件路径为空")
	}

	if _, err := os.Stat(filePath); err != nil {
		return errors.New("PathError:" + err.Error())
	}
	return nil
}

//数据切割
func SplitData(data string) (host string, port string) {
	index := strings.LastIndex(data, ".")
	host = data[:index]
	port = data[index+1:]
	return
}
