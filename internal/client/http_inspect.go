package client

import (
	"bufio"
	"bytes"
	"net/http"
)

type httpExchangeSummary struct {
	Method string
	Path   string
	Host   string

	StatusCode int

	RequestHeaders  http.Header
	ResponseHeaders http.Header
}

func summarizeHTTPExchange(reqPreview, respPreview []byte) (httpExchangeSummary, bool) {
	var s httpExchangeSummary

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(reqPreview)))
	if err != nil {
		return s, false
	}
	if req.Body != nil {
		_ = req.Body.Close()
	}

	s.Method = req.Method
	if req.URL != nil {
		s.Path = req.URL.RequestURI()
	}
	s.Host = req.Host
	s.RequestHeaders = req.Header.Clone()

	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(respPreview)), req)
	if err != nil {
		return s, false
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	s.StatusCode = resp.StatusCode
	s.ResponseHeaders = resp.Header.Clone()

	return s, true
}
