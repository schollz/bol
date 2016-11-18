package main

import (
	"encoding/json"
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

	log.Fatal(http.ListenAndServe(":9090", nil))
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello, world\n")
}

func HandlePost(w http.ResponseWriter, r *http.Request) {
	outFile, err := os.Create("test.txt")
	if err != nil {
		panic(err)
	}
	// handle err
	defer outFile.Close()
	_, err = io.Copy(outFile, r.Body)
	io.WriteString(w, "thanks\n")
}

type Result struct {
	FirstName string `json:"first"`
	LastName  string `json:"last"`
}

func HandleJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	result, _ := json.Marshal(Result{"tee", "dub"})
	io.WriteString(w, string(result))
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
