package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/schollz/bol/utils"
	"github.com/schollz/cryptopasta"
)

// https://gist.github.com/tristanwietsma/8444cf3cb5a1ac496203
type handler func(w http.ResponseWriter, r *http.Request)

func main() {
	// public views
	http.HandleFunc("/", HandleIndex)

	// private views
	http.HandleFunc("/post", PostOnly(HandlePost))
	http.HandleFunc("/pull", GetOnly(HandlePull))
	http.HandleFunc("/new", PutOnly(HandleNew))
	http.HandleFunc("/repo", DeleteOnly(HandleErase))
	fmt.Println("Running on 0.0.0.0:9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "hello, world\n")
}

func HandlePost(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	log.Println("Pushed new repo")
	username, password, _ := r.BasicAuth()
	log.Println(r.BasicAuth())
	creds := make(map[string]string)
	data, _ := ioutil.ReadFile("logins.json")
	json.Unmarshal(data, &creds)
	authenticated := false

	if passwordHash, ok := creds[username]; ok {
		if cryptopasta.CheckPasswordHash([]byte(passwordHash), []byte(password)) == nil {
			authenticated = true
		}
	} else {
		log.Println("User does not exist")
		w.WriteHeader(http.StatusNetworkAuthenticationRequired)
		io.WriteString(w, username+" does not exist")
		return
	}

	if authenticated {
		fileName := username + ".tar.bz2"
		outFile, err := os.Create(fileName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// handle err
		defer outFile.Close()
		_, err = io.Copy(outFile, r.Body)
		log.Println("Wrote file")
		io.WriteString(w, "thanks\n")
	} else {
		log.Println("Incorect password, " + password)
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "incorrect password")
	}

}

func HandleErase(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	log.Println("Erasing repo")
	username, password, _ := r.BasicAuth()
	log.Println(r.BasicAuth())
	creds := make(map[string]string)
	data, _ := ioutil.ReadFile("logins.json")
	json.Unmarshal(data, &creds)
	authenticated := false

	if passwordHash, ok := creds[username]; ok {
		if cryptopasta.CheckPasswordHash([]byte(passwordHash), []byte(password)) == nil {
			authenticated = true
		}
	} else {
		log.Println("User does not exist")
		w.WriteHeader(http.StatusNetworkAuthenticationRequired)
		io.WriteString(w, username+" does not exist")
		return
	}

	if authenticated {
		fileName := username + ".tar.bz2"
		os.Remove(fileName)
	}

}

func HandlePull(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	username, _, _ := r.BasicAuth()
	log.Println("Got repo request from " + username)
	fileName := username + ".tar.bz2"
	if utils.Exists(fileName) {
		w.Header().Set("Content-Type", "octet-stream")
		file, _ := os.Open(fileName)
		io.Copy(w, file)
	} else {
		w.WriteHeader(http.StatusNoContent)
		io.WriteString(w, "repo does not exist")
	}

}

func HandleNew(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	username, password, _ := r.BasicAuth()
	hashedPassword, _ := cryptopasta.HashPassword([]byte(password))
	creds := make(map[string]string)

	if utils.Exists("logins.json") {
		data, _ := ioutil.ReadFile("logins.json")
		json.Unmarshal(data, &creds)
		if _, ok := creds[username]; ok {
			io.WriteString(w, username+" already exists")
			return
		}
	}
	creds[username] = string(hashedPassword)
	b, _ := json.MarshalIndent(creds, "", "  ")
	ioutil.WriteFile("logins.json", b, 0644)
	io.WriteString(w, "inserted new user, "+username)

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

func DeleteOnly(h handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			h(w, r)
			return
		}
		http.Error(w, "delete only", http.StatusMethodNotAllowed)
	}
}
