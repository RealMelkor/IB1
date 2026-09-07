package web

import (
	"net/http"

	"ib1/data"
)

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(200)
	w.Write([]byte("content"))
	w.Write(data.ServeData([]byte(r.RequestURI + r.UserAgent() + r.Proto)))
}

func Listen(addr string) error {
	http.HandleFunc("/", index)
	return http.ListenAndServe(addr, nil)
}
