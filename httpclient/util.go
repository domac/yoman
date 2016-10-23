package httpclient

import (
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

//参数字符串转换
func paramsToString(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	return values.Encode()
}

func addParams(url_ string, params map[string]string) string {
	if len(params) == 0 {
		return url_
	}

	if !strings.Contains(url_, "?") {
		url_ += "?"
	}

	if strings.HasSuffix(url_, "?") || strings.HasSuffix(url_, "&") {
		url_ += paramsToString(params)
	} else {
		url_ += "&" + paramsToString(params)
	}

	return url_
}

func addFormFile(writer *multipart.Writer, name, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	part, err := writer.CreateFormFile(name, filepath.Base(path))
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)

	return err
}

func Option(o map[string]interface{}) map[int]interface{} {
	rst := make(map[int]interface{})
	for k, v := range o {
		k := "OPT_" + strings.ToUpper(k)
		if num, ok := CONST[k]; ok {
			rst[num] = v
		}
	}

	return rst
}

func mergeOptions(options ...map[int]interface{}) map[int]interface{} {
	rst := make(map[int]interface{})

	for _, m := range options {
		for k, v := range m {
			rst[k] = v
		}
	}

	return rst
}

func mergeHeaders(headers ...map[string]string) map[string]string {
	rst := make(map[string]string)

	for _, m := range headers {
		for k, v := range m {
			rst[k] = v
		}
	}

	return rst
}

func checkParamFile(params map[string]string) bool {
	for k, _ := range params {
		if k[0] == '@' {
			return true
		}
	}

	return false
}

func hasOption(opt int, options []int) bool {
	for _, v := range options {
		if opt != v {
			return true
		}
	}

	return false
}

type Map map[interface{}]interface{}

//参数解析
func parseMap(m Map) (map[int]interface{}, map[string]string) {
	var options = make(map[int]interface{})
	var headers = make(map[string]string)

	if m == nil {
		return options, headers
	}

	for k, v := range m {
		if kInt, ok := k.(int); ok {
			options[kInt] = v
		} else if kString, ok := k.(string); ok {
			kStringUpper := strings.ToUpper(kString)
			if kInt, ok := CONST[kStringUpper]; ok {
				options[kInt] = v
			} else {
				if vString, ok := v.(string); ok {
					headers[kString] = vString
				}
			}
		}
	}

	return options, headers
}
