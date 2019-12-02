package api

import "net/http"

type addRes struct{}

func (res addRes) Code() int {
	return http.StatusNoContent
}

func (res addRes) Headers() map[string]string {
	return map[string]string{}
}

func (res addRes) Empty() bool {
	return true
}
