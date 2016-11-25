package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// https://gist.github.com/tristanwietsma/8444cf3cb5a1ac496203
type handler func(w http.ResponseWriter, r *http.Request)

func BasicAuth(pass handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, _ := r.BasicAuth()
		if username != "username" || password != "password" {
			http.Error(w, "authorization failed", http.StatusUnauthorized)
			return
		}
		pass(w, r)
	}
}

func main() {
	// public views
	http.HandleFunc("/", HandleIndex)

	// private views
	http.HandleFunc("/post", PostOnly(BasicAuth(HandlePost)))
	http.HandleFunc("/json", GetOnly(BasicAuth(HandleJSON)))
	http.HandleFunc("/new", PutOnly(BasicAuth(HandleNew)))

	log.Fatal(http.ListenAndServe(":9090", nil))
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello, world\n")
}

func HandlePost(w http.ResponseWriter, r *http.Request) {
	username, _, _ := r.BasicAuth()
	outFile, err := os.Create(username + ".zip")
	if err != nil {
		panic(err)
	}
	// handle err
	defer outFile.Close()
	_, err = io.Copy(outFile, r.Body)
	fmt.Println("Wrote file")
	io.WriteString(w, "thanks\n")
}

type Result struct {
	FirstName string `json:"first"`
	LastName  string `json:"last"`
}

func HandleJSON(w http.ResponseWriter, r *http.Request) {
	username, _, _ := r.BasicAuth()
	w.Header().Set("Content-Type", "octet-stream")
	file, _ := os.Open(username + ".zip")
	io.Copy(w, file)
}

func HandleNew(w http.ResponseWriter, r *http.Request) {
	username, _, _ := r.BasicAuth()
	w.Header().Set("Content-Type", "octet-stream")
	file, _ := os.Open(username + ".zip")
	io.Copy(w, file)
}

func GetOnly(h handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			h(w, r)
			return
		}
		http.Error(w, "get only", http.StatusMethodNotAllowed)
	}
}

func PostOnly(h handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			h(w, r)
			return
		}
		http.Error(w, "post only", http.StatusMethodNotAllowed)
	}
}

func PutOnly(h handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			h(w, r)
			return
		}
		http.Error(w, "put only", http.StatusMethodNotAllowed)
	}
}
