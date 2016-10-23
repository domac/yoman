package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

type Switch struct {
	Host      string `json:"host"`
	Community string `json:"community"`
}

//从配置文件读取信息
func LoadSwitchFromFile(fileName string) ([]Switch, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	//替换JSON字符串中不合法的符号
	data = bytes.Replace(data, []byte("'"), []byte("\""), -1)

	var d []Switch
	if err = json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return d, nil
}

//从数据接口中获取交换机数据
func LoadSwitchFromUrl(url string) ([]Switch, error) {
	return nil, nil
}
