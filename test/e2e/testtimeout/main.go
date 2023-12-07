package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

type Server struct {
	timeout int
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.timeout == 0 {
		w.WriteHeader(200)
		fmt.Println("PASSING")
		return
	}
	time.Sleep(time.Duration(s.timeout * int(time.Second)))
	fmt.Println("served after waiting")
	return
}

func main() {
	fmt.Println("starting server...")
	var s Server
	if len(os.Args) > 1 && os.Args[1] == "fail" {
		s.timeout = 10
	} else {
		s.timeout = 0
	}
	http.Handle("/retry", &s)
	http.ListenAndServe(":9280", nil)
}
