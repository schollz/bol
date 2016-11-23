package ssed

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"

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

func init() {
	createDirs()
}

func createDirs() {
	dir, _ := homedir.Dir()
	pathToConfigFile = path.Join(dir, ".config", "ssed", "config.json")
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

	return &ssed{
		username: configs[0].Username,
		password: password,
		method:   configs[0].Method,
	}, nil
}
