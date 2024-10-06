package main

import (
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("./statics"))
	http.Handle("/", fs)

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./statics/login.html")
	})

	http.HandleFunc("/MobileVerify", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./statics/MobileVerify.html")
	})

	http.ListenAndServe(":8080", nil)
}
