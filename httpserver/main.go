package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func main() {
	http.Handle("/", wrapHandlerWithLogging(http.HandlerFunc(rootHandler)))
	http.Handle("/healthz", wrapHandlerWithLogging(http.HandlerFunc(healthzHandler)))
	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func rootHandler(writer http.ResponseWriter, request *http.Request) {
	message := []byte("1.接收客户端 request，并将 request 中带的 header 写入 response header:\n-------------------------\n")
	_, err := writer.Write(message)
	if err != nil {
		log.Fatal(err)
	}
	headers := request.Header
	for k, v := range headers {
		_, err := writer.Write([]byte(k + ": " + strings.Join(v, ", ") + "\n"))
		if err != nil {
			log.Fatal(err)
		}
	}

	message = []byte("\n2.读取当前系统的环境变量中的 VERSION 配置，并写入 response header:\n-------------------------\n")
	_, err = writer.Write(message)
	if err != nil {
		log.Fatal(err)
	}
	os.Setenv("VERSION", "0.0.1")
	_, err = io.WriteString(writer, "VERSION="+os.Getenv("VERSION"))
	if err != nil {
		log.Fatal(err)
	}
}

func healthzHandler(writer http.ResponseWriter, request *http.Request) {
	io.WriteString(writer, "200\n")
}

func getIp(request *http.Request) (string, error) {
	ip := request.Header.Get("X-Real-IP")
	if net.ParseIP(ip) != nil {
		return ip, nil
	}

	ips := request.Header.Get("X-Forward-For")
	for _, ip := range strings.Split(ips, ",") {
		if net.ParseIP(ip) != nil {
			return ip, nil
		}
	}

	ip, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return "", err
	}

	if net.ParseIP(ip) != nil {
		return ip, nil
	}

	return "", errors.New("No valid IP found!")
}

func wrapHandlerWithLogging(wrappedHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ip, err := getIp(req)
		if err != nil {
			log.Fatal(err)
		}
		lrw := newLoggingResponseWriter(w)
		wrappedHandler.ServeHTTP(lrw, req)

		statusCode := lrw.statusCode
		log.Printf("Client IP: %s, URL: %s ---> %d", ip, req.URL.Path, statusCode)
	})
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) writeHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
