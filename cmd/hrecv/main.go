package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

func main() {
	addr := os.Args[1]
	http.HandleFunc("/", recv)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func recv(w http.ResponseWriter, r *http.Request) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println("error:", err)
		http.Error(w, "internal error", 500)
		return
	}
	os.Stdout.Write(dump)
	os.Stdout.Write([]byte{'\n', '\n'})
}
