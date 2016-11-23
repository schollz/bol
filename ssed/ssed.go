package ssed

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

func init() {
	createDirs()
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
					return &ssed{
						username: username,
						password: password,
						method:   configs[i].Method,
					}, nil
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
	pathToSourceRepo := path.Join(pathToCacheFolder, configs[0].Username+".tar.bz2.aes")
	if utils.Exists(pathToSourceRepo) {
		key := sha256.Sum256([]byte(password))
		content, _ := ioutil.ReadFile(pathToSourceRepo)
		decrypted, err := cryptopasta.Decrypt(content, &key)
		if err != nil {
			return &ssed{}, err
		}
		ioutil.WriteFile(path.Join(pathToTempFolder, "data.tar.bz2"), decrypted, 0644)
		wd, _ := os.Getwd()
		os.Chdir(pathToTempFolder)
		archiver.TarBz2.Open("data.tar.bz2", ".")
		os.Chdir(wd)
	}

	return &ssed{
		pathToSourceRepo: pathToSourceRepo,
		username:         configs[0].Username,
		password:         password,
		method:           configs[0].Method,
	}, nil
}

// Update make a new entry
// date can be empty, it will fill in the current date if so
func (ssed ssed) Update(text, documentName, entryName, timestamp string) error {
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
	b, _ := json.MarshalIndent(entry, "", "  ")
	ioutil.WriteFile(fileName, b, 0644)
	return nil
}

// Close closes the repo and pushes if it was succesful pulling
func (ssed ssed) Close() {
	if utils.Exists(path.Join(pathToTempFolder, "data")) {
		wd, _ := os.Getwd()
		os.Chdir(pathToTempFolder)
		archiver.TarBz2.Make("data.tar.bz2", []string{"data"})
		os.Chdir(wd)
		key := sha256.Sum256([]byte(ssed.password))
		b, _ := ioutil.ReadFile(path.Join(pathToTempFolder, "data.tar.bz2"))
		encrypted, _ := cryptopasta.Encrypt(b, &key)
		ioutil.WriteFile(path.Join(pathToCacheFolder, ssed.username+".tar.bz2.aes"), encrypted, 0644)
	}
	// shred the data files
	files, _ := filepath.Glob(path.Join(pathToTempFolder, "*", "*"))
	for _, file := range files {
		utils.Shred(file)
	}
	// shred the archive
	files, _ = filepath.Glob(path.Join(pathToTempFolder, "*"))
	for _, file := range files {
		utils.Shred(file)
	}
	os.RemoveAll(pathToTempFolder)
}

type timeSlice []entry

func (p timeSlice) Len() int {
	return len(p)
}

func (p timeSlice) Less(i, j int) bool {
	return p[i].datetime.Before(p[j].datetime)
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (ssed *ssed) parseArchive() {
	files, _ := filepath.Glob(path.Join(pathToTempFolder, "data", "*"))
	ssed.entries = make(map[string]entry)
	ssed.entryNameToUUID = make(map[string]string)
	ssed.ordering = make(map[string][]string)
	var entriesToSort = make(map[string]entry)
	for _, file := range files {
		bJson, _ := ioutil.ReadFile(file)
		var entry entry
		json.Unmarshal(bJson, &entry)
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
	for _, entry := range sortedEntries {
		if val, ok := ssed.ordering[entry.Document]; !ok {
			ssed.ordering[entry.Document] = []string{entry.uuid}
		} else {
			ssed.ordering[entry.Document] = append(val, entry.uuid)
		}
	}
}
