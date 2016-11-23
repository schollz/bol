package ssed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
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
	pathToConfigFile = path.Join(dir, ".config", "ssed", "config.json")
	pathToCacheFolder = path.Join(dir, ".cache", "ssed")
	pathToTempFolder = path.Join(dir, ".cache", "ssed", "temp")
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

func ListConfigs() []config {
	b, _ := ioutil.ReadFile(pathToConfigFile)
	var configs []config
	json.Unmarshal(b, &configs)
	return configs
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

type ssed struct {
	parsed           bool
	shredding        bool
	pathToSourceRepo string
	username         string
	password         string
	method           string
	entries          map[string]entry    // uuid -> entry
	entryNameToUUID  map[string]string   // entry name -> uuid
	ordering         map[string][]string // document -> list of entry uuids in order
}

func (ssed ssed) ReturnMethod() string {
	return ssed.method
}

func (ssed *ssed) SetMethod(method string) error {
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
	ioutil.WriteFile(pathToConfigFile, b, 0644)
	return nil
}

// Open allows the user to initialize a new filesystem
// The username and password is used to authenticate
// If the username is blank, it uses the first current user
// The method cannot be blank for a new user
// The method cannot be overridden with open, it must be overridden with SetMethod
func Open(username, password, method string) (*ssed, error) {
	defer timeTrack(time.Now(), "Opening archive")
	createDirs()
	var configs []config
	hashedPassword, _ := cryptopasta.HashPassword([]byte(password))
	if !utils.Exists(pathToConfigFile) {
		// Configuration file doesn't exists, create it
		if !strings.Contains(method, "http") && !strings.Contains(method, "ssh") {
			return &ssed{}, errors.New("Incorrect method provided")
		}
		if len(username) == 0 {
			return &ssed{}, errors.New("Must provide user name")
		}
		configs = []config{
			config{
				Username:       username,
				HashedPassword: string(hashedPassword),
				Method:         method,
			},
		}
	} else {
		// Configuration file already exists
		// If the user exists, verify password and continue
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
				if cryptopasta.CheckPasswordHash([]byte(configs[i].HashedPassword), []byte(password)) != nil {
					return &ssed{}, errors.New("Incorrect password")
				} else {
					method = configs[i].Method
				}
				foundConfig = i
				break
			}
		}
		if foundConfig == -1 {
			// configuration is old, but is added to the front as it will be the new default
			if !strings.Contains(method, "http") && !strings.Contains(method, "ssh") {
				return &ssed{}, errors.New("Incorrect method provided")
			}
			configs = append([]config{config{
				Username:       username,
				HashedPassword: string(hashedPassword),
				Method:         method,
			}}, configs...)
		} else {
			// configuration is old, but will add to front to be new default
			currentConfig := configs[foundConfig]
			otherConfigs := append(configs[:foundConfig], configs[foundConfig+1:]...)
			configs = append([]config{currentConfig}, otherConfigs...)
		}
	}

	b, _ := json.MarshalIndent(configs, "", "  ")
	ioutil.WriteFile(pathToConfigFile, b, 0644)

	// open repo
	sourceRepo := path.Join(pathToCacheFolder, configs[0].Username+".tar.bz2")
	if utils.Exists(sourceRepo) {
		utils.CopyFile(sourceRepo, path.Join(pathToTempFolder, "data.tar.bz2"))
		wd, _ := os.Getwd()
		os.Chdir(pathToTempFolder)
		defer timeTrack(time.Now(), "Unzipping")
		archiver.TarBz2.Open("data.tar.bz2", ".")
		os.Chdir(wd)
	}

	return &ssed{
		pathToSourceRepo: sourceRepo,
		username:         configs[0].Username,
		password:         password,
		method:           configs[0].Method,
	}, nil
}

// Update make a new entry
// date can be empty, it will fill in the current date if so
func (ssed *ssed) Update(text, documentName, entryName, timestamp string) error {
	if !utils.Exists(path.Join(pathToTempFolder, "data")) {
		os.Mkdir(path.Join(pathToTempFolder, "data"), 0755)
	}
	fileName := path.Join(pathToTempFolder, "data", utils.HashAndHex(text+"file contents")+".json")
	if len(entryName) == 0 {
		entryName = utils.RandStringBytesMaskImprSrc(10)
	}
	if len(timestamp) == 0 {
		timestamp = time.Now().Format(time.RFC3339)
	}

	entry := entry{
		Text:      text,
		Document:  documentName,
		Entry:     entryName,
		Timestamp: timestamp,
	}
	b, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	key := sha256.Sum256([]byte(ssed.password))
	encrypted, _ := cryptopasta.Encrypt(b, &key)

	err = ioutil.WriteFile(fileName, []byte(hex.EncodeToString(encrypted)), 0644)
	ssed.parsed = false
	return err
}

// Delete Entry will simply Update("ignore-entry",documentName,entryName,"")
func (ssed *ssed) DeleteEntry(documentName, entryName string) {
	ssed.Update("ignore entry", documentName, entryName, "")
	ssed.parsed = false
}

// Delete Entry will simply Update("ignore document",documentName,entryName,"")
func (ssed *ssed) DeleteDocument(documentName string) {
	ssed.Update("ignore document", documentName, "", "")
	ssed.parsed = false
}

// Close closes the repo and pushes if it was succesful pulling
func (ssed ssed) Close() {
	defer timeTrack(time.Now(), "Closing archive")
	if utils.Exists(path.Join(pathToTempFolder, "data")) {
		wd, _ := os.Getwd()
		os.Chdir(pathToTempFolder)
		archiver.TarBz2.Make(ssed.pathToSourceRepo, []string{"data"})
		os.Chdir(wd)
	}
	// shred the data files
	if ssed.shredding {
		files, _ := filepath.Glob(path.Join(pathToTempFolder, "*", "*"))
		for _, file := range files {
			err := utils.Shred(file)
			if err == nil {
				// logger.Debug("Shredded %s", file)
			}
		}
		// shred the archive
		files, _ = filepath.Glob(path.Join(pathToTempFolder, "*"))
		for _, file := range files {
			err := utils.Shred(file)
			if err == nil {
				// logger.Debug("Shredded %s", file)
			}
		}
	}
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

func (ssed *ssed) parseArchive() {
	defer timeTrack(time.Now(), "Parsing archive")
	files, _ := filepath.Glob(path.Join(pathToTempFolder, "data", "*"))
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
		json.Unmarshal(decrypted, &entry)
		entry.uuid = file
		entry.datetime, _ = time.Parse(time.RFC3339, entry.Timestamp)
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

func (ssed *ssed) ListDocuments() []string {
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

func (ssed *ssed) GetDocument(documentName string) []entry {
	defer timeTrack(time.Now(), "Getting document "+documentName)
	if !ssed.parsed {
		ssed.parseArchive()
	}
	entries := make([]entry, len(ssed.ordering[documentName]))
	curEntry := 0
	for _, uuid := range ssed.ordering[documentName] {
		if ssed.entries[uuid].Text == "ignore document" {
			return []entry{}
		}
		if ssed.entries[uuid].Text == "ignore entry" {
			continue
		}
		entries[curEntry] = ssed.entries[uuid]
		curEntry++
	}
	return entries[0:curEntry]
}

func (ssed *ssed) GetEntry(documentName, entryName string) (entry, error) {
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
