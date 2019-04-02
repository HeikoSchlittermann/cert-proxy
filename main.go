package main

import (
	"fmt"
	"log"
	"net/http"
)

func fatal(err error) {
	if err == nil {
		return
	}
	log.Fatal(err)
}

func welcome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Header)
	if r.Header["Api-Key"] == nil || r.Header["Api-Key"][0] != "foo" {
	    w.WriteHeader(401)
	    w.Write([]byte("Missing API-Key header"))
	    return
	}
	fmt.Fprintf(w, "Welcome\r\n")
}

func main() {

	http.HandleFunc("/", welcome)
	err := http.ListenAndServeTLS(":4433", "crt.pem", "crt.pem", nil)
	fatal(err)

}
