package ssed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jcelliott/lumber"
	"github.com/mholt/archiver"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/schollz/bol/utils"
	"github.com/schollz/cryptopasta"
)

// Generic functions

type config struct {
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
	Method         string `json:"method"`
}

var pathToConfigFile string
var pathToCacheFolder string
var pathToTempFolder string
var pathToLocalFolder string
var pathToRemoteFolder string
var homePath string
var logger *lumber.ConsoleLogger

func DebugMode() {
	logger.Level(0)
}

func init() {
	logger = lumber.NewConsoleLogger(lumber.DEBUG)
	logger.Level(2)
}

func createDirs() {
	dir, _ := homedir.Dir()
	homePath = dir
	if !utils.Exists(path.Join(dir, ".config")) {
		os.MkdirAll(path.Join(dir, ".config"), 0755)
	}
	if !utils.Exists(path.Join(dir, ".config", "ssed")) {
		os.MkdirAll(path.Join(dir, ".config", "ssed"), 0755)
	}
	if !utils.Exists(path.Join(dir, ".cache")) {
		os.MkdirAll(path.Join(dir, ".cache"), 0755)
	}
	if !utils.Exists(path.Join(dir, ".cache", "ssed")) {
		os.MkdirAll(path.Join(dir, ".cache", "ssed"), 0755)
	}
	if !utils.Exists(path.Join(dir, ".cache", "ssed", "temp")) {
		os.MkdirAll(path.Join(dir, ".cache", "ssed", "temp"), 0755)
	}
	if !utils.Exists(path.Join(dir, ".cache", "ssed", "local")) {
		os.MkdirAll(path.Join(dir, ".cache", "ssed", "local"), 0755)
	}
	if !utils.Exists(path.Join(dir, ".cache", "ssed", "remote")) {
		os.MkdirAll(path.Join(dir, ".cache", "ssed", "remote"), 0755)
	}
	pathToConfigFile = path.Join(dir, ".config", "ssed", "config.json")
	pathToCacheFolder = path.Join(dir, ".cache", "ssed")
	pathToTempFolder = path.Join(dir, ".cache", "ssed", "temp")
	pathToLocalFolder = path.Join(dir, ".cache", "ssed", "local")
	pathToRemoteFolder = path.Join(dir, ".cache", "ssed", "remote")
}

// Filesystem specific

type entry struct {
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	Document  string `json:"document"`
	Entry     string `json:"entry"`
	datetime  time.Time
	uuid      string
}

type document struct {
	Entries []entry
}

type Fs struct {
	wg               sync.WaitGroup
	havePassword     bool
	parsed           bool
	shredding        bool
	pathToSourceRepo string
	pathToLocalRepo  string
	pathToRemoteRepo string
	username         string
	password         string
	method           string
	archiveName      string
	entries          map[string]entry    // uuid -> entry
	entryNameToUUID  map[string]string   // entry name -> uuid
	ordering         map[string][]string // document -> list of entry uuids in order
}

func EraseConfig() {
	utils.Shred(pathToConfigFile)
}

func EraseAll() {
	createDirs()
	EraseConfig()
	os.RemoveAll(pathToCacheFolder)
	os.RemoveAll(pathToConfigFile)
	files, _ := filepath.Glob(path.Join(pathToCacheFolder, "*"))
	for _, file := range files {
		fmt.Println(file)
	}
}

// CleanUp shreds all the temporary files
func CleanUp() {
	files, _ := filepath.Glob(path.Join(pathToTempFolder, "*"))
	for _, file := range files {
		logger.Debug("Shredding %s", file)
		utils.Shred(file)
	}
}

func ListConfigs() []config {
	b, _ := ioutil.ReadFile(pathToConfigFile)
	var configs []config
	json.Unmarshal(b, &configs)
	return configs
}

// ReturnMethod returns the current method being used
func (ssed Fs) ReturnMethod() string {
	return ssed.method
}

// ReturnUser returns the current user being used
func (ssed Fs) ReturnUser() string {
	return ssed.username
}

func (ssed *Fs) SetMethod(method string) error {
	if !strings.Contains(method, "http") && !strings.Contains(method, "ssh") {
		return errors.New("Incorrect method provided")
	}
	ssed.method = method
	b, _ := ioutil.ReadFile(pathToConfigFile)
	var configs []config
	json.Unmarshal(b, &configs)
	for i := range configs {
		if configs[i].Username == ssed.username {
			// check if password matches
			configs[i].Method = method
			break
		}
	}
	b, _ = json.MarshalIndent(configs, "", "  ")
	ioutil.WriteFile(pathToConfigFile, b, 0755)
	return nil
}

// Init initializes the repo
// If the username and the method are left blank it will automatically use first
// found in the config file
func (ssed *Fs) Init(username, method string) error {
	defer timeTrack(time.Now(), "Opening archive")
	createDirs()
	if len(method) > 0 && !strings.Contains(method, "http") && !strings.Contains(method, "ssh") {
		return errors.New("Method must be http or ssh")
	}
	// Load configuration
	err := ssed.loadConfiguration(username, method)
	if err != nil {
		return err
	}

	// create nessecary folders if nessecary
	ssed.pathToLocalRepo = path.Join(pathToCacheFolder, "local", ssed.username)
	if !utils.Exists(ssed.pathToLocalRepo) {
		os.MkdirAll(ssed.pathToLocalRepo, 0755)
	}
	ssed.pathToRemoteRepo = path.Join(pathToCacheFolder, "remote", ssed.username)
	if !utils.Exists(ssed.pathToRemoteRepo) {
		os.MkdirAll(ssed.pathToRemoteRepo, 0755)
	}

	// download and decompress asynchronously
	ssed.wg = sync.WaitGroup{}
	ssed.wg.Add(1)
	go ssed.downloadAndDecompress()
	return nil
}

func (ssed *Fs) loadConfiguration(username, method string) error {
	var configs []config
	if !utils.Exists(pathToConfigFile) {
		if len(username) == 0 {
			return errors.New("Need to have username to intialize for first time")
		}
		// Configuration file doesn't exists, create it
		configs = []config{
			config{
				Username: username,
				Method:   method,
			},
		}
	} else {
		// Configuration file already exists
		// If the user does not exists, add user as new default and continue
		b, _ := ioutil.ReadFile(pathToConfigFile)
		json.Unmarshal(b, &configs)
		foundConfig := -1
		for i := range configs {
			// uses the default (first entry)
			if username == "" {
				username = configs[i].Username
			}
			if configs[i].Username == username {
				// check if password matches
				method = configs[i].Method
				foundConfig = i
				break
			}
		}
		if foundConfig == -1 {
			// configuration is new, and is added to the front as it will be the new default
			configs = append([]config{config{
				Username: username,
				Method:   method,
			}}, configs...)
		} else {
			// configuration is old, but will add to front to be new default
			currentConfig := configs[foundConfig]
			otherConfigs := append(configs[:foundConfig], configs[foundConfig+1:]...)
			configs = append([]config{currentConfig}, otherConfigs...)
		}
	}

	b, _ := json.MarshalIndent(configs, "", "  ")
	ioutil.WriteFile(pathToConfigFile, b, 0755)

	ssed.method = configs[0].Method
	ssed.username = configs[0].Username
	ssed.archiveName = ssed.username + ".tar.bz2"
	return nil
}

func (ssed *Fs) downloadAndDecompress() {
	// download repo
	if strings.Contains(ssed.method, "http") {
		defer timeTrack(time.Now(), "download")
		req, err := http.NewRequest("GET", ssed.method+"/pull", nil)
		if err != nil {
			panic(err)
		}
		req.SetBasicAuth(ssed.username, "") // no password needed

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		outFile, err := os.Create(path.Join(pathToRemoteFolder, ssed.archiveName))
		if err != nil {
			panic(err)
		}
		// handle err
		defer outFile.Close()
		_, err = io.Copy(outFile, resp.Body)

		// open remote repo
		if utils.Exists(path.Join(pathToRemoteFolder, ssed.archiveName)) {
			wd, _ := os.Getwd()
			os.Chdir(pathToRemoteFolder)
			logger.Debug("Opening remote")
			archiver.TarBz2.Open(ssed.archiveName, ".")
			os.Chdir(wd)
		} else {
			os.Mkdir(path.Join(pathToRemoteFolder, ssed.username), 0755)
		}
	}

	// open local repo
	sourceRepo := path.Join(pathToLocalFolder, ssed.archiveName)
	if utils.Exists(sourceRepo) {
		wd, _ := os.Getwd()
		os.Chdir(pathToLocalFolder)
		logger.Debug("Opening local")
		archiver.TarBz2.Open(ssed.archiveName, ".")
		os.Chdir(wd)
	} else {
		os.Mkdir(path.Join(pathToLocalFolder, ssed.username), 0755)
	}
	logger.Debug("Download Finished")
	ssed.wg.Done()

	// copy over files
	ssed.copyOverFiles()

}

func (ssed Fs) copyOverFiles() {
	localFiles := make(map[string]bool)
	files, _ := filepath.Glob(path.Join(pathToLocalFolder, ssed.username, "*"))
	for _, file := range files {
		localFiles[filepath.Base(file)] = true
	}

	files, _ = filepath.Glob(path.Join(pathToRemoteFolder, ssed.username, "*"))
	for _, file := range files {
		if _, ok := localFiles[filepath.Base(file)]; !ok {
			// local doesn't have this remote file! copy it over
			utils.CopyFile(path.Join(pathToRemoteFolder, ssed.username, filepath.Base(file)), path.Join(pathToLocalFolder, ssed.username, filepath.Base(file)))
			logger.Debug("Copying over " + filepath.Base(file))
		}
	}

}

func openAndDecrypt(filename string, password string) (string, error) {
	key := sha256.Sum256([]byte(password))
	content, err := ioutil.ReadFile(filename)
	contentData, err := hex.DecodeString(string(content))
	if err != nil {
		return "", err
	}
	decrypted, err := cryptopasta.Decrypt(contentData, &key)
	return string(decrypted), err
}

func (ssed *Fs) Open(password string) error {
	// only continue if the downloading is finished
	ssed.wg.Wait()
	logger.Debug("Finished waiting")

	// check password against one of the files (if they exist)
	files, _ := filepath.Glob(path.Join(pathToLocalFolder, ssed.username, "*"))
	for _, file := range files {
		logger.Debug("Testing against %s", file)
		_, err := openAndDecrypt(file, password)
		if err != nil {
			return err
		} else {
			break
		}
	}
	ssed.password = password
	return nil
}

// Update make a new entry
// date can be empty, it will fill in the current date if so
func (ssed *Fs) Update(text, documentName, entryName, timestamp string) error {
	fileName := path.Join(ssed.pathToLocalRepo, utils.HashAndHex(text+"file contents")+".json")
	if len(entryName) == 0 {
		entryName = utils.RandStringBytesMaskImprSrc(10)
	}
	if len(timestamp) == 0 {
		timestamp = utils.GetCurrentDate()
	} else {
		timestamp = utils.ReFormatDate(timestamp)
	}

	e := entry{
		Text:      text,
		Document:  documentName,
		Entry:     entryName,
		Timestamp: timestamp,
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}

	key := sha256.Sum256([]byte(ssed.password))
	encrypted, _ := cryptopasta.Encrypt(b, &key)

	err = ioutil.WriteFile(fileName, []byte(hex.EncodeToString(encrypted)), 0755)
	ssed.parsed = false
	return err
}

// Delete Entry will simply Update("ignore-entry",documentName,entryName,"")
func (ssed *Fs) DeleteEntry(documentName, entryName string) {
	ssed.Update("ignore entry", documentName, entryName, "")
	ssed.parsed = false
}

// Delete Entry will simply Update("ignore document",documentName,entryName,"")
func (ssed *Fs) DeleteDocument(documentName string) {
	ssed.Update("ignore document", documentName, "", "")
	ssed.parsed = false
}

// Close closes the repo and pushes if it was succesful pulling
func (ssed Fs) Close() {
	defer timeTrack(time.Now(), "Closing archive")
	if strings.Contains(ssed.method, "http") {
		logger.Debug("Archiving")
		wd, _ := os.Getwd()
		os.Chdir(pathToLocalFolder)
		archiver.TarBz2.Make(ssed.archiveName, []string{ssed.username})

		logger.Debug("Pushing")
		// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
		file, err := os.Open(ssed.archiveName)
		defer file.Close()
		req, err := http.NewRequest("POST", ssed.method+"/post", file)
		if err != nil {
			panic(err)
		}
		req.SetBasicAuth(ssed.username, ssed.password)
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(os.Stdout, resp.Body)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		os.Chdir(wd)
	}

	// // shred the data files
	// if ssed.shredding {
	// 	files, _ := filepath.Glob(path.Join(pathToTempFolder, "*", "*"))
	// 	for _, file := range files {
	// 		err := utils.Shred(file)
	// 		if err == nil {
	// 			// logger.Debug("Shredded %s", file)
	// 		}
	// 	}
	// 	// shred the archive
	// 	files, _ = filepath.Glob(path.Join(pathToTempFolder, "*"))
	// 	for _, file := range files {
	// 		err := utils.Shred(file)
	// 		if err == nil {
	// 			// logger.Debug("Shredded %s", file)
	// 		}
	// 	}
	// }
	os.RemoveAll(pathToTempFolder)
}

type timeSlice []entry

func (p timeSlice) Len() int {
	return len(p)
}

// need to sort in reverse-chronological order to get only newest
func (p timeSlice) Less(i, j int) bool {
	return p[j].datetime.Before(p[i].datetime)
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (ssed *Fs) parseArchive() {
	defer timeTrack(time.Now(), "Parsing archive")
	files, _ := filepath.Glob(path.Join(ssed.pathToLocalRepo, "*"))
	ssed.entries = make(map[string]entry)
	ssed.entryNameToUUID = make(map[string]string)
	ssed.ordering = make(map[string][]string)
	var entriesToSort = make(map[string]entry)
	for _, file := range files {
		key := sha256.Sum256([]byte(ssed.password))
		content, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		contentData, err := hex.DecodeString(string(content))
		if err != nil {
			panic(err)
		}
		decrypted, err := cryptopasta.Decrypt(contentData, &key)
		if err != nil {
			panic(err)
		}
		var entry entry
		err = json.Unmarshal(decrypted, &entry)
		if err != nil {
			panic(err)
		}
		entry.uuid = file
		entry.datetime, _ = utils.ParseDate(entry.Timestamp)
		ssed.entries[file] = entry
		ssed.entryNameToUUID[entry.Entry] = file
		entriesToSort[file] = entry
	}

	sortedEntries := make(timeSlice, 0, len(entriesToSort))
	for _, d := range entriesToSort {
		sortedEntries = append(sortedEntries, d)
	}
	sort.Sort(sortedEntries)
	alreadyAddedEntry := make(map[string]bool)
	for _, entry := range sortedEntries {
		if _, ok := alreadyAddedEntry[entry.Entry]; ok {
			continue
		}
		if val, ok := ssed.ordering[entry.Document]; !ok {
			ssed.ordering[entry.Document] = []string{entry.uuid}
		} else {
			ssed.ordering[entry.Document] = append(val, entry.uuid)
		}
		alreadyAddedEntry[entry.Entry] = true
	}
	// go in chronological order, so reverse list
	for key := range ssed.ordering {
		for i, j := 0, len(ssed.ordering[key])-1; i < j; i, j = i+1, j-1 {
			ssed.ordering[key][i], ssed.ordering[key][j] = ssed.ordering[key][j], ssed.ordering[key][i]
		}
	}
	ssed.parsed = true
}

func (ssed *Fs) ListEntries() []string {
	if !ssed.parsed {
		ssed.parseArchive()
	}
	entries := make([]string, len(ssed.entryNameToUUID))
	i := 0
	for entry := range ssed.entryNameToUUID {
		entries[i] = entry
		i++
	}
	return entries
}

func (ssed *Fs) ListDocuments() []string {
	defer timeTrack(time.Now(), "Listing documents")
	if !ssed.parsed {
		ssed.parseArchive()
	}
	documents := []string{}
	for document := range ssed.ordering {
		ignoring := false
		for _, uuid := range ssed.ordering[document] {
			if ssed.entries[uuid].Text == "ignore document" {
				ignoring = true
				break
			}
		}
		if !ignoring {
			documents = append(documents, document)
		}
	}
	return documents
}

func (ssed *Fs) GetDocument(documentName string) []entry {
	defer timeTrack(time.Now(), "Getting document "+documentName)
	if !ssed.parsed {
		ssed.parseArchive()
	}
	entries := make([]entry, len(ssed.ordering[documentName]))
	curEntry := 0
	for _, uuid := range ssed.ordering[documentName] {
		if ssed.entries[uuid].Text == "ignore document" {
			logger.Debug("Ignoring document %s", ssed.entries[uuid].Timestamp)
			return []entry{}
		}
		if ssed.entries[uuid].Text == "ignore entry" {
			logger.Debug("Ignoring entry %s", ssed.entries[uuid].Timestamp)
			continue
		}
		entries[curEntry] = ssed.entries[uuid]
		curEntry++
	}
	return entries[0:curEntry]
}

func (ssed *Fs) GetEntry(documentName, entryName string) (entry, error) {
	defer timeTrack(time.Now(), "Getting entry "+entryName)
	if !ssed.parsed {
		ssed.parseArchive()
	}

	var entry entry
	for _, uuid := range ssed.ordering[documentName] {
		if ssed.entries[uuid].Entry == entryName {
			if ssed.entries[uuid].Text == "ignore entry" {
				return entry, errors.New("Entry deleted")
			} else {
				return ssed.entries[uuid], nil
			}
		}
	}
	return entry, errors.New("Entry not found")
}

// timeTrack from https://coderwall.com/p/cp5fya/measuring-execution-time-in-go
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	logger.Debug("%s took %s", name, elapsed)
}
