package middleware

import (
	"net/http"
)

type ResponseCapture struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func NewResponseCapture(w http.ResponseWriter) *ResponseCapture {
	return &ResponseCapture{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

func (c *ResponseCapture) Status() int {
	return c.status
}

func (c *ResponseCapture) BytesWritten() int {
	return c.bytes
}

func (c *ResponseCapture) WroteHeader() bool {
	return c.wroteHeader
}

func (c *ResponseCapture) WriteHeader(status int) {
	if c.wroteHeader {
		return
	}

	c.wroteHeader = true
	c.status = status

	c.ResponseWriter.WriteHeader(status)
}

func (c *ResponseCapture) Write(body []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}

	n, err := c.ResponseWriter.Write(body)
	c.bytes += n

	return n, err
}

func (c *ResponseCapture) Unwrap() http.ResponseWriter {
	return c.ResponseWriter
}
