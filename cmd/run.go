package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/jcelliott/lumber"
	"github.com/kardianos/osext"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
	ssed "github.com/schollz/bol/ssed"
	"github.com/schollz/bol/utils"
)

var logger *lumber.ConsoleLogger
var homePath string
var JOURNAL_DELIMITER = "///"

func DebugMode() {
	logger.Level(0)
	ssed.DebugMode()
}

func init() {
	logger = lumber.NewConsoleLogger(lumber.DEBUG)
	logger.Level(2)
	homePath, _ = homedir.Dir()
	if !utils.Exists(path.Join(homePath, ".cache")) {
		os.MkdirAll(path.Join(homePath, ".cache"), 0755)
	}
	if !utils.Exists(path.Join(homePath, ".cache", "ssed")) {
		os.MkdirAll(path.Join(homePath, ".cache", "ssed"), 0755)
	}
	if !utils.Exists(path.Join(homePath, ".cache", "ssed", "temp")) {
		os.MkdirAll(path.Join(homePath, ".cache", "ssed", "temp"), 0755)
	}
}

func Run(workingFile string, changeUser bool) {
	var fs ssed.Fs
	var err error
	err = fs.Init("", "")
	if err != nil || changeUser {
		var username, method string
		fmt.Print("Enter username: ")
		fmt.Scanln(&username)
		method = "https://bol.schollz.com"
		fs.Init(username, method)
		//err2 := fs.Init(username, "https://bol.schollz.com") // should raise error if it doesn't exist
		// if err2 != nil {
		// 	fmt.Print("Enter method (default https://bol.schollz.com): ")
		// 	fmt.Scanln(&method)
		// 	fs.Init(username, method)
		// }
	} else {
		fmt.Println(fs.ReturnUser(), fs.ReturnMethod())
	}
	for {
		fmt.Printf("Enter password: ")
		var password string
		if runtime.GOOS == "windows" {
			fmt.Scanln(&password) // not great fix, but works for cygwin
		} else {
			bytePassword, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
			password = strings.TrimSpace(string(bytePassword))
		}
		err = fs.Open(password)
		if err == nil {
			break
		} else {
			fmt.Println("Incorrect password.")
		}
	}
	defer fs.Close()

	entries := ssed.GetBlankEntries()
	isNewEntry := true
	logger.Debug("Working file input: '%s'", workingFile)
	if len(workingFile) == 0 {
		data := [][]string{}
		for fileNum, file := range fs.ListDocuments() {
			data = append(data, []string{strconv.Itoa(fileNum + 1), file})
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"#", "Document"})
		for _, v := range data {
			table.Append(v)
		}
		fmt.Printf("\n")
		table.Render()

		var document string
		fmt.Print("Enter document: ")
		fmt.Scanln(&document)
		document = strings.TrimSpace(document)
		if i, err := strconv.Atoi(document); err == nil {
			workingFile = data[i-1][1]
		} else {
			workingFile = document
		}
		fmt.Printf("Opening %s\n", workingFile)
		entries = fs.GetDocument(workingFile)
	} else {
		logger.Debug("Parsing whether it is a document or entry")
		entries, isNewEntry, workingFile, _ = fs.GetDocumentOrEntry(workingFile)
	}

	fullText := ""
	for _, entry := range entries {
		fullText += fmt.Sprintf("%s %s\n%s\n\n%s\n\n", JOURNAL_DELIMITER, entry.Entry, entry.Timestamp, strings.TrimSpace(entry.Text))
	}
	if isNewEntry {
		fullText += fmt.Sprintf("%s %s\n%s\n\n\n%s", JOURNAL_DELIMITER, utils.GetRandomMD5Hash(), utils.GetCurrentDate(), "")
	}

	newText := WriteEntry(fullText, "vim")
	for _, splitText := range strings.Split(newText, JOURNAL_DELIMITER) {
		lines := strings.Split(splitText, "\n")
		if len(lines) < 3 {
			continue
		}
		newEntryText := strings.TrimSpace(strings.Join(lines[2:], "\n"))
		if len(newEntryText) == 0 {
			continue
		}
		entryName := ""
		if len(lines[0]) > 1 {
			entryName = strings.TrimSpace(lines[0])
		}
		timestamp := strings.TrimSpace(lines[1])
		fs.Update(newEntryText, workingFile, entryName, timestamp)
	}
}

func WriteEntry(text string, editor string) string {
	logger.Debug("Editing file")

	ioutil.WriteFile(path.Join(ssed.PathToTempFolder, "temp"), []byte(text), 0644)

	var cmdArgs []string
	if editor == "vim" {
		// Setup vim
		vimrc := `set nocompatible
set backspace=2
func! WordProcessorModeCLI()
	setlocal formatoptions=t1
	setlocal textwidth=80
	map j gj
	map k gk
	set formatprg=par
	setlocal wrap
	setlocal linebreak
	setlocal noexpandtab
	normal G$
endfu
com! WPCLI call WordProcessorModeCLI()`
		// Append to .vimrc file
		if utils.Exists(path.Join(ssed.PathToTempFolder, ".vimrc")) {
			// Check if .vimrc file contains code
			logger.Debug("Found .vimrc.")
			fileContents, err := ioutil.ReadFile(path.Join(ssed.PathToTempFolder, ".vimrc"))
			if err != nil {
				log.Fatal(err)
			}
			if !strings.Contains(string(fileContents), "com! WPCLI call WordProcessorModeCLI") {
				// Append to fileContents
				logger.Debug("WPCLI not found in .vimrc, adding it...")
				newvimrc := string(fileContents) + "\n" + vimrc
				err := ioutil.WriteFile(path.Join(ssed.PathToTempFolder, ".vimrc"), []byte(newvimrc), 0644)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				logger.Debug("WPCLI found in .vimrc.")
			}
		} else {
			logger.Debug("Can not find .vimrc, creating new .vimrc...")
			err := ioutil.WriteFile(path.Join(ssed.PathToTempFolder, ".vimrc"), []byte(vimrc), 0644)
			if err != nil {
				log.Fatal(err)
			}
		}

		cmdArgs = []string{"-u", path.Join(ssed.PathToTempFolder, ".vimrc"), "-c", "WPCLI", "+startinsert", path.Join(ssed.PathToTempFolder, "temp")}

	} else if editor == "nano" {
		lines := "100" // TODO: DETERMINE THIS
		cmdArgs = []string{"+" + lines + ",1000000", "-r", "80", "--tempfile", path.Join(ssed.PathToTempFolder, "temp")}
	} else if editor == "emacs" {
		lines := "100" // TODO: DETERMINE THIS
		cmdArgs = []string{"+" + lines + ":1000000", path.Join(ssed.PathToTempFolder, "temp")}
	} else if editor == "micro" {
		settings := `{
    "autoclose": false,
    "autoindent": false,
    "colorscheme": "zenburn",
    "cursorline": false,
    "gofmt": false,
    "goimports": false,
    "ignorecase": false,
    "indentchar": " ",
    "linter": false,
    "ruler": false,
    "savecursor": false,
    "saveundo": false,
    "scrollmargin": 3,
    "scrollspeed": 2,
    "statusline": false,
    "syntax": false,
    "tabsize": 4,
    "tabstospaces": false,
		"softwrap": true
}`
		if !utils.Exists(path.Join(homePath, ".config", "micro")) {
			os.MkdirAll(path.Join(homePath, ".config", "micro"), 0755)
		}
		err := ioutil.WriteFile(path.Join(homePath, ".config", "micro", "settings.json"), []byte(settings), 0644)
		if err != nil {
			log.Fatal(err)
		}

		lines := "10000000" // TODO determine this
		cmdArgs = []string{"-startpos", lines + ",1000000", path.Join(ssed.PathToTempFolder, "temp")}
	}

	extension := ""
	if runtime.GOOS == "windows" {
		extension = ".exe"
	}

	// Load from binary assets
	logger.Debug("Trying to get asset: %s", "bin/"+editor+extension)
	data, err := Asset("bin/" + editor + extension)
	if err == nil {
		logger.Debug("Using builtin editor: %s", "bin/"+editor+extension)
		err = ioutil.WriteFile(path.Join(homePath, ".cache", "ssed", "temp", editor+extension), data, 0755)
		if err != nil {
			log.Fatal(err)
		}
		editor = path.Join(homePath, ".cache", "ssed", "temp", editor)
	} else {
		logger.Debug("Could not find builtin editor: %s", err.Error())
	}

	// Run the editor
	logger.Debug("Using editor %s", editor+extension)
	cmd := exec.Command(editor+extension, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		logger.Debug("Failed using builtin")
		// Try to execute from the same folder
		logger.Debug("Error running: %s", err.Error())
		programPath, _ := osext.ExecutableFolder()
		editor = filepath.Base(editor)
		cmd := exec.Command(path.Join(programPath, editor+extension), cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err2 := cmd.Run()
		if err2 != nil {
			logger.Debug("Failed using from same folder as executable")
			// Try to execute from system path
			logger.Debug("Error running: %s", err.Error())
			cmd := exec.Command(editor+extension, cmdArgs...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			err3 := cmd.Run()
			if err3 != nil {
				logger.Debug("Failed using from system path")
				log.Fatal(err3)
			}
		}
	}
	fileContents, _ := ioutil.ReadFile(path.Join(ssed.PathToTempFolder, "temp"))
	return strings.TrimSpace(string(fileContents))
}
