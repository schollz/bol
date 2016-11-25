package ssed

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

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
	var fs ssed
	EraseConfig()
	dir, _ := homedir.Dir()
	configFile := path.Join(dir, ".config", "ssed", "config.json")

	fs.Init("zack", "ssh://server1")
	fs.Open("test")
	if !utils.Exists(configFile) {
		t.Errorf("Problem creating configuation file")
	}
	fs.Init("zack", "ssh://server1")
	fs.Open("test")
	if fs.ReturnMethod() != "ssh://server1" {
		t.Errorf("Problem reloading method")
	}

	// Test the setting and getting of methods in memory
	firstMethod := fs.ReturnMethod()
	fs.SetMethod("http://someothermethod")
	secondMethod := fs.ReturnMethod()
	if firstMethod != "ssh://server1" || secondMethod != "http://someothermethod" {
		t.Errorf("Problem using pointers in structs")
	}

}

func TestEntries(t *testing.T) {
	var text string
	var fs ssed
	EraseAll()
	// Test adding a entry
	// DebugMode()
	fs.Init("zack", "ssh://server1")
	fs.Open("test")
	fs.Update("some text", "notes", "", "2014-11-20T13:00:00-05:00")
	fs.Update("some other test", "journal", "", "2014-11-20T13:00:00-05:00")
	fs.Update("some other test", "journal", "getEntry", "2010-11-20T13:00:00-05:00")
	fs.Update("some text2", "notes", "", "2015-11-23T13:00:00-05:00")
	fs.Update("some text3", "notes", "entry1", "2016-11-20T13:00:00-05:00")
	fs.Update("some text4", "notes", "entry2", "2013-11-20T13:00:00-05:00")
	fs.Update("some text3, edited", "notes", "entry1", "2016-11-23T13:00:00-05:00")
	// for i := 0; i < 1000; i++ {
	// 	text := strconv.Itoa(i)
	// 	fs.Update("asdf laksdfj alskdj flaks jdflkas jdfl"+text, "test", text, "")
	// }

	// check if ordering is correct
	text = ""
	for _, entry := range fs.GetDocument("notes") {
		text += fmt.Sprintln(entry.Document, entry.Timestamp, entry.Text)
	}
	fs.Close()
	if text != `notes 2013-11-20T13:00:00-05:00 some text4
notes 2014-11-20T13:00:00-05:00 some text
notes 2015-11-23T13:00:00-05:00 some text2
notes 2016-11-23T13:00:00-05:00 some text3, edited
` {
		t.Errorf("Ordering is not correct!")
	}

	// check if deletion of entry works
	fs.Init("zack", "")
	fs.Open("test")
	fs.DeleteEntry("notes", "entry2")
	text = ""
	for _, entry := range fs.GetDocument("notes") {
		text += fmt.Sprintln(entry.Document, entry.Timestamp, entry.Text)
	}
	fs.Close()
	if text != `notes 2014-11-20T13:00:00-05:00 some text
notes 2015-11-23T13:00:00-05:00 some text2
notes 2016-11-23T13:00:00-05:00 some text3, edited
` {
		t.Errorf("Deleting entry did not work: '%s'", text)
	}

	// check if deletion of document works
	fs.Init("zack", "ssh://server1")
	fs.Open("test")
	text = fmt.Sprintln(fs.ListDocuments())
	if text != "[notes journal]\n" {
		t.Errorf("Initial listing of documents is wrong")
	}
	fs.DeleteDocument("notes")
	text = fmt.Sprintln(fs.ListDocuments())
	if text != "[journal]\n" {
		t.Errorf("Listing of documents is wrong after deletion")
	}
	text = ""
	for _, entry := range fs.GetDocument("notes") {
		text += fmt.Sprintln(entry.Document, entry.Timestamp, entry.Text)
	}
	fs.Close()
	if text != `` {
		t.Errorf("Document should be empty after deletion")
	}

	fs.Init("zack", "ssh://server1")
	fs.Open("test")
	entry, _ := fs.GetEntry("journal", "getEntry")
	text = strings.TrimSpace(fmt.Sprintln(entry.Document, entry.Timestamp, entry.Text))
	fs.Close()
	if text != `journal 2010-11-20T13:00:00-05:00 some other test` {
		t.Errorf("Problem loading single entry")
	}

	// fs2, _ := Open("zack2", "test2", "http://something")
	// fs2.Update("blah", "texts", "", "2014-11-21T13:00:00-05:00")
	// fs2.Update("ghjgjgj", "texts", "", "2014-11-20T13:00:00-05:00")
	// // check if ordering is correct
	// for _, entry := range fs2.GetDocument("texts") {
	// 	fmt.Println(entry.Document, entry.Timestamp, entry.Text)
	// }
	// fs2.Close()

}

func TestOpen(t *testing.T) {
	var fs ssed
	fs.Init("zack", "http://test")
	fs.Open("password")
}
