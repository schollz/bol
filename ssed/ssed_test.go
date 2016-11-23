package ssed

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/schollz/bol/utils"
)

func TestCreateDirs(t *testing.T) {
	dir, _ := homedir.Dir()
	os.RemoveAll(path.Join(dir, ".config", "ssed"))
	os.RemoveAll(path.Join(dir, ".config", "cache"))
	createDirs()
	if !utils.Exists(path.Join(dir, ".config", "ssed")) || !utils.Exists(path.Join(dir, ".cache", "ssed")) {
		t.Errorf("Problem creating dirs")
	}
}

func TestConfig(t *testing.T) {
	EraseConfig()
	dir, _ := homedir.Dir()
	configFile := path.Join(dir, ".config", "ssed", "config.json")
	Open("zack", "test", "ssh://server1")
	if !utils.Exists(configFile) {
		t.Errorf("Problem creating configuation file")
	}
	fs, _ := Open("zack", "test", "")
	if fs.ReturnMethod() != "ssh://server1" {
		t.Errorf("Problem reloading method")
	}
	_, err := Open("zack", "wrongpassword", "")
	if err == nil {
		t.Errorf("Should have an error")
	}

	// Test the setting and getting of methods in memory
	firstMethod := fs.ReturnMethod()
	fs.SetMethod("http://someothermethod")
	secondMethod := fs.ReturnMethod()
	if firstMethod != "ssh://server1" || secondMethod != "http://someothermethod" {
		t.Errorf("Problem using pointers in structs")
	}
	// Test setting bad method
	err = fs.SetMethod("badmethod")
	if err == nil {
		t.Errorf("Error should be thrown for bad method")
	}
	// Test getting changed method from disk
	fs2, _ := Open("zack", "test", "")
	if fs2.ReturnMethod() != "http://someothermethod" {
		t.Errorf("Problem with persistence of method")
	}

	// Test loading the default user with corret password
	fs2, _ = Open("", "test", "")
	if fs2.username != "zack" {
		t.Errorf("Could not load default user")
	}
	// Test loading the default user with incorrect password
	_, err = Open("", "tesjkljlt", "")
	if err == nil {
		t.Errorf("Problem with password")
	}

	// Test listing configs
	Open("zack2", "test2", "ssh://server2")
	configs := ListConfigs()
	if configs[0].Username != "zack2" || configs[1].Username != "zack" {
		t.Errorf("Error setting configs: %+v", configs) // last name should be listed first
	}

}

func TestEntries(t *testing.T) {
	// Test adding a entry
	fs, _ := Open("zack", "test", "")
	defer fs.Close()
	fs.Update("some text", "notes", "", "")
	time.Sleep(1000 * time.Millisecond)
	fs.Update("some text2", "notes", "", "")
	time.Sleep(1000 * time.Millisecond)
	fs.Update("some text3", "notes", "", "")
	fs.parseArchive()

	// check if ordering is correct
	for _, uuid := range fs.ordering["notes"] {
		fmt.Println(fs.entries[uuid].Text, fs.entries[uuid].Timestamp)
	}
}
