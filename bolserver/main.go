package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schollz/bol/ssed"

	"github.com/schollz/bol/utils"
	"github.com/schollz/cryptopasta"
)

// https://gist.github.com/tristanwietsma/8444cf3cb5a1ac496203
type handler func(w http.ResponseWriter, r *http.Request)

var wd string
var apikeys = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

var MaxArchiveBytes int
var Port, Host string
var (
	Version, BuildTime, Build, OS, LastCommit string
)

func main() {
	flag.IntVar(&MaxArchiveBytes, "limit", 10000000, "limit the max size (bytes) of archive with backups")
	flag.StringVar(&Port, "port", "9095", "set port")
	flag.StringVar(&Host, "host", "", "set hostname")
	flag.Parse()
	wd, _ = os.Getwd()
	http.HandleFunc("/", HandleLogin)
	http.HandleFunc("/login", HandleLoginAttempt)
	http.HandleFunc("/register", HandleRegisterAttempt)
	http.HandleFunc("/post", HandlePostAttempt)
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		data, err := Asset(r.URL.Path[1:])
		if err != nil {
			log.Println("Error finding asset")
		}
		w.Write(data)
		// http.ServeFile(w, r, r.URL.Path[1:])
	})
	http.HandleFunc("/repo", HandleRepo) // POST latest repo
	if Host != "" {
		fmt.Printf("Running on http://%s:%s, aliased as %s\n", GetLocalIP(), Port, Host)
	} else {
		fmt.Printf("Running on http://%s:%s\n", GetLocalIP(), Port)
	}
	fmt.Printf("Saving up to %d MB for archives\n", MaxArchiveBytes/1000000)
	log.Fatal(http.ListenAndServe(":"+Port, nil))
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	ShowLoginPage(w, r, "", "")
}

type loginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func HandlePostAttempt(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ShowLoginPage(w, r, "Problem handling post", "danger")
		return
	}
	if !strings.Contains(string(body), "=") {
		ShowLoginPage(w, r, "You need to login first", "info")
		return
	}
	lines := strings.Split(string(body), "=")
	apikey := strings.TrimSpace(strings.Split(lines[1], "document")[0])
	document := strings.TrimSpace(strings.Split(lines[2], "entry")[0])
	entry := strings.TrimSpace(strings.Split(lines[3], "data")[0])
	text := strings.Join(strings.Split(strings.TrimSpace(lines[4]), "\r"), "")
	var username, password string
	apikeys.Lock()
	if val, ok := apikeys.m[apikey]; ok {
		username = strings.Split(val, "=")[0]
		password = strings.Split(val, "=")[1]
	} else {
		apikeys.Unlock()
		ShowLoginPage(w, r, "Incorrect API key", "danger")
		return
	}
	delete(apikeys.m, apikey)
	apikeys.Unlock()
	fmt.Printf("document:%s\ntext:%s\n", document, text)
	go updateRepo(username, password, text, document, entry, "")
	ShowLoginPage(w, r, "Updated entry", "success")
}

func deleteApikeyDelay(apikey string) {
	time.Sleep(30 * time.Minute)
	log.Printf("Deleting apikey: %s\n", apikey)
	apikeys.Lock()
	delete(apikeys.m, apikey)
	apikeys.Unlock()
}

func updateRepo(username, password, text, document, entry, date string) {
	var fs ssed.Fs
	fs.Init(username, "http://127.0.0.1:"+Port)
	fs.Open(password)
	fs.Update(text, document, entry, date)
	fs.Close()
}

func ShowLoginPage(w http.ResponseWriter, r *http.Request, message string, messageType string) {
	messageHTML := `<div class="col-xs-12"><div class="alert alert-` + messageType + `">
  ` + message + `
</div></div>`
	if len(message) == 0 {
		messageHTML = ""
	}
	page, err := Asset("login.html")
	if err != nil {
		log.Println("Error finding asset")
	}
	pageS := string(page)
	pageS = strings.Replace(pageS, "MESSAGE", messageHTML, -1)
	fmt.Fprintf(w, "%s", pageS)
}

func HandleRegisterAttempt(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ShowLoginPage(w, r, "Error reading body", "danger")
		return
	}
	if !strings.Contains(string(body), "&") {
		ShowLoginPage(w, r, "Bad login attempt", "danger")
		return
	}
	data := strings.Split(string(body), "&")
	username := strings.TrimSpace(strings.Split(data[0], "=")[1])
	password := strings.TrimSpace(strings.Split(data[1], "=")[1])
	if len(username) == 0 && len(password) == 0 {
		ShowLoginPage(w, r, "Username and password must not be empty", "danger")
		return
	}

	hashedPassword, _ := cryptopasta.HashPassword([]byte(password))
	creds := make(map[string]string)

	if utils.Exists(path.Join(wd, "logins.json")) {
		data, _ := ioutil.ReadFile(path.Join(wd, "logins.json"))
		json.Unmarshal(data, &creds)
		if _, ok := creds[username]; ok {
			ShowLoginPage(w, r, "User '"+username+"' already exists", "info")
			return
		}
	}
	creds[username] = string(hashedPassword)
	b, _ := json.MarshalIndent(creds, "", "  ")
	ioutil.WriteFile(path.Join(wd, "logins.json"), b, 0644)
	ShowLoginPage(w, r, "Added user '"+username+"'", "success")
}

func HandleLoginAttempt(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ShowLoginPage(w, r, "Error reading body", "danger")
		return
	}
	if !strings.Contains(string(body), "&") {
		ShowLoginPage(w, r, "Bad login attempt", "danger")
		return
	}
	data := strings.Split(string(body), "&")
	username := strings.TrimSpace(strings.Split(data[0], "=")[1])
	password := strings.TrimSpace(strings.Split(data[1], "=")[1])
	creds := make(map[string]string)
	loginData, _ := ioutil.ReadFile(path.Join(wd, "logins.json"))
	json.Unmarshal(loginData, &creds)
	authenticated := false

	if passwordHash, ok := creds[username]; ok {
		if cryptopasta.CheckPasswordHash([]byte(passwordHash), []byte(password)) == nil {
			authenticated = true
		}
	} else {
		ShowLoginPage(w, r, "User "+username+" does not exist", "info")
		return
	}

	if authenticated {
		page, err := Asset("post.html")
		if err != nil {
			log.Println("Error finding asset")
		}
		pageS := string(page)
		apikey := utils.RandStringBytesMaskImprSrc(30)
		apikeys.Lock()
		apikeys.m[apikey] = username + "=" + password
		apikeys.Unlock()
		pageS = strings.Replace(pageS, "keyXX", apikey, -1)
		var fs ssed.Fs
		fs.Init(username, "http://127.0.0.1:"+Port)
		fs.Open(password)
		documentList := fs.ListDocuments()
		if len(documentList) == 0 {
			documentList = append(documentList, "notes")
		}
		newHTML := "<option>" + strings.Join(documentList, "</option><option>") + "</option>"
		fs.Close()
		pageS = strings.Replace(pageS, "OPTIONS", newHTML, -1)

		fmt.Fprintf(w, "%s", pageS)
		go deleteApikeyDelay(apikey)
	} else {
		ShowLoginPage(w, r, "Incorrect password", "info")
	}
}

func HandlePush(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	username, password, _ := r.BasicAuth()
	log.Printf("Got repo request for %s\n", username)
	creds := make(map[string]string)
	data, _ := ioutil.ReadFile(path.Join(wd, "logins.json"))
	json.Unmarshal(data, &creds)

	authenticated := false

	if passwordHash, ok := creds[username]; ok {
		if cryptopasta.CheckPasswordHash([]byte(passwordHash), []byte(password)) == nil {
			authenticated = true
			log.Printf("Authentication success for %s\n", username)
		}
	} else {
		log.Printf("PUSH: User '%s' does not exist\n", username)
		w.WriteHeader(http.StatusNetworkAuthenticationRequired)
		io.WriteString(w, username+" does not exist, goto "+Host+" to register user")
		return
	}

	if authenticated {
		initializeUser(username)
		fileName := path.Join(wd, "archive", username, username+"."+utils.GetUnixTimestamp()+".tar.bz2")

		outFile, err := os.Create(fileName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// handle err
		defer outFile.Close()
		_, err = io.Copy(outFile, r.Body)
		go cleanFiles(username)
		log.Printf("PUSH: Wrote file '%s' for '%s'\n", fileName, username)
		io.WriteString(w, string(fmt.Sprintf("Wrote file for '%s' on %s\n", username, Host)))
	} else {
		log.Println("Incorect password for" + username)
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "incorrect password")
	}

}

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	log.Println("Erasing repo")
	username, password, _ := r.BasicAuth()
	creds := make(map[string]string)
	data, _ := ioutil.ReadFile(path.Join(wd, "logins.json"))
	json.Unmarshal(data, &creds)
	authenticated := false

	if passwordHash, ok := creds[username]; ok {
		if cryptopasta.CheckPasswordHash([]byte(passwordHash), []byte(password)) == nil {
			authenticated = true
		}
	} else {
		log.Printf("DELETE: User '%s' does not exist", username)
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
	latestFileName, err := getLatestFileName(username)
	if err == nil {
		w.Header().Set("Content-Type", "octet-stream")
		file, _ := os.Open(path.Join(wd, "archive", username, latestFileName))
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

	if utils.Exists(path.Join(wd, "logins.json")) {
		data, _ := ioutil.ReadFile(path.Join(wd, "logins.json"))
		json.Unmarshal(data, &creds)
		if _, ok := creds[username]; ok {
			io.WriteString(w, username+" already exists")
			return
		}
	}
	creds[username] = string(hashedPassword)
	b, _ := json.MarshalIndent(creds, "", "  ")
	ioutil.WriteFile(path.Join(wd, "logins.json"), b, 0644)
	io.WriteString(w, "inserted new user, "+username)
}

func HandleRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		HandlePull(w, r)
	} else if r.Method == "POST" {
		HandlePush(w, r)
	} else if r.Method == "DELETE" {
		HandleDelete(w, r)
	} else if r.Method == "PUT" {
		HandleNew(w, r)
	}
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

func initializeUser(username string) {
	pathToData := path.Join(wd, "archive", username)
	log.Println(pathToData)
	if !utils.Exists(path.Join(wd, "archive")) {
		os.Mkdir(path.Join(wd, "archive"), 0755)
	}
	if !utils.Exists(path.Join(wd, "archive", username)) {
		os.Mkdir(path.Join(wd, "archive", username), 0755)
	}
}

func getLatestFileName(username string) (string, error) {
	files, _ := ioutil.ReadDir(path.Join(wd, "archive", username))
	if len(files) == 0 {
		return "", errors.New("No files yet")
	}
	timestamps := make([]int, len(files))
	for i, f := range files {
		s := strings.Split(f.Name(), ".")
		timestamp, err := strconv.Atoi(s[1])
		if err == nil {
			timestamps[i] = timestamp
		}
	}
	sort.Ints(timestamps)
	return fmt.Sprintf("%s.%s.tar.bz2", username, strconv.Itoa(timestamps[len(timestamps)-1])), nil
}

func cleanFiles(username string) {
	files, _ := ioutil.ReadDir(path.Join(wd, "archive", username))
	if len(files) == 0 {
		return
	}

	timestamps := make([]int, len(files))
	for i, f := range files {
		s := strings.Split(f.Name(), ".")
		timestamp, err := strconv.Atoi(s[1])
		if err == nil {
			timestamps[i] = timestamp
		}
	}

	filenames := make([]string, len(timestamps))
	for i, timestamp := range timestamps {
		filenames[i] = path.Join(wd, "archive", username, fmt.Sprintf("%s.%s.tar.bz2", username, strconv.Itoa(timestamp)))
	}

	removeI := 0
	for {
		totalSize := int64(0)
		for _, filename := range filenames {
			fi, e := os.Stat(filename)
			if e != nil {
				continue
			}
			// get the size
			totalSize += fi.Size()
		}
		log.Printf("Total size of %s archive is %d", username, totalSize)

		if totalSize > int64(MaxArchiveBytes) && len(filenames)-removeI > 1 {
			log.Printf("Removing file %s", filenames[removeI])
			os.Remove(filenames[removeI])
			removeI++
		} else {
			break
		}
	}
}

// GetLocalIP returns the local ip address
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}
	bestIP := "localhost"
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil && (strings.Contains(ipnet.IP.String(), "192.168.1") || strings.Contains(ipnet.IP.String(), "192.168")) {
				return ipnet.IP.String()
			}
		}
	}
	return bestIP
}
