package httpclient

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	VERSION   = "1.0"
	USERAGENT = "yoman httpclient v" + VERSION

	PROXY_HTTP            = 0
	PROXY_SOCKS4          = 4
	PROXY_SOCKS5          = 5
	PROXY_SOCKS4A         = 6
	OPT_AUTOREFERER       = 58
	OPT_FOLLOWLOCATION    = 52
	OPT_CONNECTTIMEOUT    = 78
	OPT_CONNECTTIMEOUT_MS = 156
	OPT_MAXREDIRS         = 68
	OPT_PROXYTYPE         = 101
	OPT_TIMEOUT           = 13
	OPT_TIMEOUT_MS        = 155
	OPT_COOKIEJAR         = 10082
	OPT_INTERFACE         = 10062
	OPT_PROXY             = 10004
	OPT_REFERER           = 10016
	OPT_USERAGENT         = 10018
	OPT_REDIRECT_POLICY   = 100000
	OPT_PROXY_FUNC        = 100001
	OPT_DEBUG             = 100002
)

var CONST = map[string]int{
	"OPT_AUTOREFERER":       58,
	"OPT_FOLLOWLOCATION":    52,
	"OPT_CONNECTTIMEOUT":    78,
	"OPT_CONNECTTIMEOUT_MS": 156,
	"OPT_MAXREDIRS":         68,
	"OPT_PROXYTYPE":         101,
	"OPT_TIMEOUT":           13,
	"OPT_TIMEOUT_MS":        155,
	"OPT_COOKIEJAR":         10082,
	"OPT_INTERFACE":         10062,
	"OPT_PROXY":             10004,
	"OPT_REFERER":           10016,
	"OPT_USERAGENT":         10018,
	"OPT_REDIRECT_POLICY":   100000,
	"OPT_PROXY_FUNC":        100001,
	"OPT_DEBUG":             100002,
}

// 默认选项
var defaultOptions = map[int]interface{}{
	OPT_FOLLOWLOCATION: true,
	OPT_MAXREDIRS:      10,
	OPT_AUTOREFERER:    true,
	OPT_USERAGENT:      USERAGENT,
	OPT_COOKIEJAR:      true,
	OPT_DEBUG:          false,
}

var transportOptions = []int{
	OPT_CONNECTTIMEOUT,
	OPT_CONNECTTIMEOUT_MS,
	OPT_PROXYTYPE,
	OPT_TIMEOUT,
	OPT_TIMEOUT_MS,
	OPT_INTERFACE,
	OPT_PROXY,
	OPT_PROXY_FUNC,
}

var jarOptions = []int{
	OPT_COOKIEJAR,
}

// 封装 http.Response
type Response struct {
	*http.Response
}

//读取全部请求数据
func (this *Response) ReadAll() ([]byte, error) {
	var reader io.ReadCloser
	var err error
	switch this.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(this.Body)
		if err != nil {
			return nil, err
		}
	default:
		reader = this.Body
	}

	defer reader.Close()
	return ioutil.ReadAll(reader)
}

func (this *Response) ToString() (string, error) {
	bytes, err := this.ReadAll()
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// 请求预处理
func prepareRequest(method string, url_ string, headers map[string]string, body io.Reader, options map[int]interface{}) (*http.Request, error) {
	req, err := http.NewRequest(method, url_, body)

	if err != nil {
		return nil, err
	}

	if referer, ok := options[OPT_REFERER]; ok {
		if refererStr, ok := referer.(string); ok {
			req.Header.Set("Referer", refererStr)
		}
	}

	if useragent, ok := options[OPT_USERAGENT]; ok {
		if useragentStr, ok := useragent.(string); ok {
			req.Header.Set("User-Agent", useragentStr)
		}
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

//传输预处理
func prepareTransport(options map[int]interface{}) (http.RoundTripper, error) {
	transport := &http.Transport{}

	connectTimeoutMS := 0

	if connectTimeoutMS_, ok := options[OPT_CONNECTTIMEOUT_MS]; ok {
		if connectTimeoutMS, ok = connectTimeoutMS_.(int); !ok {
			return nil, fmt.Errorf("OPT_CONNECTTIMEOUT_MS must be int")
		}
	} else if connectTimeout_, ok := options[OPT_CONNECTTIMEOUT]; ok {
		if connectTimeout, ok := connectTimeout_.(int); ok {
			connectTimeoutMS = connectTimeout * 1000
		} else {
			return nil, fmt.Errorf("OPT_CONNECTTIMEOUT must be int")
		}
	}

	timeoutMS := 0

	if timeoutMS_, ok := options[OPT_TIMEOUT_MS]; ok {
		if timeoutMS, ok = timeoutMS_.(int); !ok {
			return nil, fmt.Errorf("OPT_TIMEOUT_MS must be int")
		}
	} else if timeout_, ok := options[OPT_TIMEOUT]; ok {
		if timeout, ok := timeout_.(int); ok {
			timeoutMS = timeout * 1000
		} else {
			return nil, fmt.Errorf("OPT_TIMEOUT must be int")
		}
	}

	if timeoutMS > 0 && (connectTimeoutMS > timeoutMS || connectTimeoutMS == 0) {
		connectTimeoutMS = timeoutMS
	}

	transport.Dial = func(network, addr string) (net.Conn, error) {
		var conn net.Conn
		var err error
		if connectTimeoutMS > 0 {
			conn, err = net.DialTimeout(network, addr, time.Duration(connectTimeoutMS)*time.Millisecond)
			if err != nil {
				return nil, err
			}
		} else {
			conn, err = net.Dial(network, addr)
			if err != nil {
				return nil, err
			}
		}

		if timeoutMS > 0 {
			conn.SetDeadline(time.Now().Add(time.Duration(timeoutMS) * time.Millisecond))
		}

		return conn, nil
	}

	// proxy
	if proxyFunc_, ok := options[OPT_PROXY_FUNC]; ok {
		if proxyFunc, ok := proxyFunc_.(func(*http.Request) (int, string, error)); ok {
			transport.Proxy = func(req *http.Request) (*url.URL, error) {
				proxyType, u_, err := proxyFunc(req)
				if err != nil {
					return nil, err
				}

				if proxyType != PROXY_HTTP {
					return nil, fmt.Errorf("only PROXY_HTTP is currently supported")
				}

				u_ = "http://" + u_

				u, err := url.Parse(u_)

				if err != nil {
					return nil, err
				}

				return u, nil
			}
		} else {
			return nil, fmt.Errorf("OPT_PROXY_FUNC is not a desired function")
		}
	} else {
		var proxytype int
		if proxytype_, ok := options[OPT_PROXYTYPE]; ok {
			if proxytype, ok = proxytype_.(int); !ok || proxytype != PROXY_HTTP {
				return nil, fmt.Errorf("OPT_PROXYTYPE must be int, and only PROXY_HTTP is currently supported")
			}
		}

		var proxy string
		if proxy_, ok := options[OPT_PROXY]; ok {
			if proxy, ok = proxy_.(string); !ok {
				return nil, fmt.Errorf("OPT_PROXY must be string")
			}
			proxy = "http://" + proxy
			proxyUrl, err := url.Parse(proxy)
			if err != nil {
				return nil, err
			}
			transport.Proxy = http.ProxyURL(proxyUrl)
		}
	}

	return transport, nil
}

func prepareRedirect(options map[int]interface{}) (func(req *http.Request, via []*http.Request) error, error) {
	var redirectPolicy func(req *http.Request, via []*http.Request) error

	if redirectPolicy_, ok := options[OPT_REDIRECT_POLICY]; ok {
		if redirectPolicy, ok = redirectPolicy_.(func(*http.Request, []*http.Request) error); !ok {
			return nil, fmt.Errorf("OPT_REDIRECT_POLICY is not a desired function")
		}
	} else {
		var followlocation bool
		if followlocation_, ok := options[OPT_FOLLOWLOCATION]; ok {
			if followlocation, ok = followlocation_.(bool); !ok {
				return nil, fmt.Errorf("OPT_FOLLOWLOCATION must be bool")
			}
		}

		var maxredirs int
		if maxredirs_, ok := options[OPT_MAXREDIRS]; ok {
			if maxredirs, ok = maxredirs_.(int); !ok {
				return nil, fmt.Errorf("OPT_MAXREDIRS must be int")
			}
		}

		redirectPolicy = func(req *http.Request, via []*http.Request) error {
			// no follow
			if !followlocation || maxredirs <= 0 {
				return &Error{
					Code:    ERR_REDIRECT_POLICY,
					Message: fmt.Sprintf("redirect not allowed"),
				}
			}

			if len(via) >= maxredirs {
				return &Error{
					Code:    ERR_REDIRECT_POLICY,
					Message: fmt.Sprintf("stopped after %d redirects", len(via)),
				}
			}

			last := via[len(via)-1]
			if useragent := last.Header.Get("User-Agent"); useragent != "" {
				req.Header.Set("User-Agent", useragent)
			}

			return nil
		}
	}

	return redirectPolicy, nil
}

func prepareJar(options map[int]interface{}) (http.CookieJar, error) {
	var jar http.CookieJar
	var err error
	if optCookieJar_, ok := options[OPT_COOKIEJAR]; ok {
		if optCookieJar, ok := optCookieJar_.(bool); ok {
			if optCookieJar {
				jar, err = cookiejar.New(nil)
				if err != nil {
					return nil, err
				}
			}
		} else if optCookieJar, ok := optCookieJar_.(http.CookieJar); ok {
			jar = optCookieJar
		} else {
			return nil, fmt.Errorf("invalid cookiejar")
		}
	}

	return jar, nil
}

//创建HTTP客户端
func NewHttpClient() *HttpClient {
	c := &HttpClient{
		reuseTransport: true,
		reuseJar:       true,
	}

	return c
}

type HttpClient struct {
	Options        map[int]interface{}
	Headers        map[string]string
	oneTimeOptions map[int]interface{}
	oneTimeHeaders map[string]string
	oneTimeCookies []*http.Cookie
	transport      http.RoundTripper
	jar            http.CookieJar
	reuseTransport bool
	reuseJar       bool
	lock           *sync.Mutex
}

func (this *HttpClient) Defaults(defaults Map) *HttpClient {
	options, headers := parseMap(defaults)

	if this.Options == nil {
		this.Options = options
	} else {
		for k, v := range options {
			this.Options[k] = v
		}
	}

	if this.Headers == nil {
		this.Headers = headers
	} else {
		for k, v := range headers {
			this.Headers[k] = v
		}
	}

	return this
}

func (this *HttpClient) Begin() *HttpClient {
	if this.lock == nil {
		this.lock = new(sync.Mutex)
	}
	this.lock.Lock()

	return this
}

func (this *HttpClient) reset() {
	this.oneTimeOptions = nil
	this.oneTimeHeaders = nil
	this.oneTimeCookies = nil
	this.reuseTransport = true
	this.reuseJar = true
	if this.lock != nil {
		this.lock.Unlock()
	}
}

func (this *HttpClient) WithOption(k int, v interface{}) *HttpClient {
	if this.oneTimeOptions == nil {
		this.oneTimeOptions = make(map[int]interface{})
	}
	this.oneTimeOptions[k] = v

	if !hasOption(k, transportOptions) {
		this.reuseTransport = false
	}

	if !hasOption(k, jarOptions) {
		this.reuseJar = false
	}

	return this
}

func (this *HttpClient) WithOptions(m Map) *HttpClient {
	options, _ := parseMap(m)
	for k, v := range options {
		this.WithOption(k, v)
	}

	return this
}

func (this *HttpClient) WithHeader(k string, v string) *HttpClient {
	if this.oneTimeHeaders == nil {
		this.oneTimeHeaders = make(map[string]string)
	}
	this.oneTimeHeaders[k] = v

	return this
}

func (this *HttpClient) WithHeaders(m map[string]string) *HttpClient {
	for k, v := range m {
		this.WithHeader(k, v)
	}

	return this
}

func (this *HttpClient) WithCookie(cookies ...*http.Cookie) *HttpClient {
	this.oneTimeCookies = append(this.oneTimeCookies, cookies...)

	return this
}

//核心调用方法
func (this *HttpClient) Do(method string, url string, headers map[string]string, body io.Reader) (*Response, error) {
	options := mergeOptions(defaultOptions, this.Options, this.oneTimeOptions)
	headers = mergeHeaders(this.Headers, this.oneTimeHeaders, headers)
	cookies := this.oneTimeCookies

	var transport http.RoundTripper
	var jar http.CookieJar
	var err error

	if this.transport == nil || !this.reuseTransport {
		transport, err = prepareTransport(options)
		if err != nil {
			this.reset()
			return nil, err
		}

		if this.reuseTransport {
			this.transport = transport
		}
	} else {
		transport = this.transport
	}

	// jar
	if this.jar == nil || !this.reuseJar {
		jar, err = prepareJar(options)
		if err != nil {
			this.reset()
			return nil, err
		}

		if this.reuseJar {
			this.jar = jar
		}
	} else {
		jar = this.jar
	}

	// 释放锁
	this.reset()

	redirect, err := prepareRedirect(options)
	if err != nil {
		return nil, err
	}

	c := &http.Client{
		Transport:     transport,
		CheckRedirect: redirect,
		Jar:           jar,
	}

	req, err := prepareRequest(method, url, headers, body, options)
	if err != nil {
		return nil, err
	}
	if debugEnabled, ok := options[OPT_DEBUG]; ok {
		if debugEnabled.(bool) {
			dump, err := httputil.DumpRequestOut(req, true)
			if err == nil {
				fmt.Printf("%s\n", dump)
			}
		}
	}

	if jar != nil {
		jar.SetCookies(req.URL, cookies)
	} else {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
	}

	res, err := c.Do(req)

	return &Response{res}, err
}

// HEAD 请求
func (this *HttpClient) Head(url string, params map[string]string) (*Response, error) {
	url = addParams(url, params)

	return this.Do("HEAD", url, nil, nil)
}

func (this *HttpClient) Get(url string, params map[string]string) (*Response, error) {
	url = addParams(url, params)

	return this.Do("GET", url, nil, nil)
}

func (this *HttpClient) Delete(url string, params map[string]string) (*Response, error) {
	url = addParams(url, params)

	return this.Do("DELETE", url, nil, nil)
}

func (this *HttpClient) Post(url string, params map[string]string) (*Response, error) {
	if checkParamFile(params) {
		return this.PostMultipart(url, params)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	body := strings.NewReader(paramsToString(params))

	return this.Do("POST", url, headers, body)
}

func (this *HttpClient) PostMultipart(url string, params map[string]string) (
	*Response, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 检查文件
	for k, v := range params {
		// 是文件的情况
		if k[0] == '@' {
			err := addFormFile(writer, k[1:], v)
			if err != nil {
				return nil, err
			}
		} else {
			writer.WriteField(k, v)
		}
	}
	headers := make(map[string]string)

	headers["Content-Type"] = writer.FormDataContentType()
	err := writer.Close()
	if err != nil {
		return nil, err
	}

	return this.Do("POST", url, headers, body)
}

func (this *HttpClient) Cookies(url_ string) []*http.Cookie {
	if this.jar != nil {
		u, _ := url.Parse(url_)
		return this.jar.Cookies(u)
	}

	return nil
}

func (this *HttpClient) CookieValues(url_ string) map[string]string {
	m := make(map[string]string)

	for _, c := range this.Cookies(url_) {
		m[c.Name] = c.Value
	}

	return m
}

func (this *HttpClient) CookieValue(url_ string, key string) string {
	for _, c := range this.Cookies(url_) {
		if c.Name == key {
			return c.Value
		}
	}

	return ""
}
