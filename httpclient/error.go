package httpclient

import (
	"fmt"
	"net"
	"strings"
)

// 错误类型
const (
	_ = iota
	ERR_DEFAULT
	ERR_TIMEOUT
	ERR_REDIRECT_POLICY
)

type Error struct {
	Code    int
	Message string
}

func (this Error) Error() string {
	return fmt.Sprintf("httpclient #%d: %s", this.Code, this.Message)
}

func getErrorCode(err error) int {
	if err == nil {
		return 0
	}

	if e, ok := err.(*Error); ok {
		return e.Code
	}

	return ERR_DEFAULT
}

func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(net.Error); ok && e.Timeout() {
		return true
	}

	if strings.Contains(err.Error(), "timeout") {
		return true
	}

	return false
}

func IsRedirectError(err error) bool {
	if err == nil {
		return false
	}

	if getErrorCode(err) == ERR_REDIRECT_POLICY {
		return true
	}

	if strings.Contains(err.Error(), "redirect") {
		return true
	}

	return false
}
