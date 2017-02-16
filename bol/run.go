package main

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

	"github.com/fatih/color"
	"github.com/jcelliott/lumber"
	"github.com/kardianos/osext"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
	ssed "github.com/schollz/bol/ssed"
	"github.com/schollz/bol/utils"
	"github.com/schollz/quotation-explorer/getquote"
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
	logger.Level(0)
	logger.Debug("Initializing")
	homePath, _ = homedir.Dir()
	if !utils.Exists(path.Join(homePath, ".config")) {
		os.MkdirAll(path.Join(homePath, ".config"), 0755)
	}
	if !utils.Exists(path.Join(homePath, ".config", "bol")) {
		c := color.New(color.FgCyan)
		c.Println("Welcome to bol!")
		fmt.Println("")
		os.MkdirAll(path.Join(homePath, ".config", "bol"), 0755)
	}
}

func Run(workingFile string, changeUser bool, dumpFile bool) {
	var fs ssed.Fs
	var err error
	err = fs.Init("", "")
	if err != nil || changeUser {
		var username, method string
		fmt.Print("Enter username: ")
		fmt.Scanln(&username)
		fmt.Print("Enter server (blank for https://bol.schollz.com): ")
		fmt.Scanln(&method)
		if len(method) == 0 {
			method = "https://bol.schollz.com"
		} else {
			method = strings.TrimSpace(method)
		}
		fs.Init(username, method)
		//err2 := fs.Init(username, "https://bol.schollz.com") // should raise error if it doesn't exist
		// if err2 != nil {
		// 	fmt.Print("Enter method (default https://bol.schollz.com): ")
		// 	fmt.Scanln(&method)
		// 	fs.Init(username, method)
		// }
	} else {
		c := color.New(color.FgHiCyan)
		fmt.Print("User:\t")
		c.Println(fs.ReturnUser())
		fmt.Print("Server:\t")
		c.Println(fs.ReturnMethod())
	}
	for {
		password := utils.GetPassword()
		err = fs.Open(password)
		if err == nil {
			// Check user status
			_, err2 := utils.CreateBolUser(fs.ReturnUser(), password, fs.ReturnMethod())
			if err2 != nil {
				c := color.New(color.FgCyan)
				c.Printf("\n\n%s\n", "Cannot connect to server, working locally")
			}
			break
		} else {
			fmt.Println("Incorrect password.")
		}
	}
	defer func() {
		err := fs.Close()
		if dumpFile {
			return
		}
		if err != nil {
			c := color.New(color.FgCyan)
			c.Printf("\nUpdated local copy of '%s'\n", workingFile)
		} else {
			c := color.New(color.FgCyan)
			c.Printf("\nUploaded changes to '%s'\n", workingFile)
		}
	}()
	if dumpFile {
		filename, _ := fs.DumpAll()
		fmt.Printf("\nContents written to %s\nRead using bol --decrypt %s\n\n", filename, filename)
		return
	}

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
		if len(workingFile) == 0 {
			workingFile = "notes"
		}
		quote := getquote.GetQuote()
		if len(quote) > 0 {
			c := color.New(color.FgGreen)
			c.Printf("\n\n%s\n\n", getquote.GetQuote())
		}
		entries = fs.GetDocument(workingFile)
	} else {
		logger.Debug("Parsing whether it is a document or entry")
		entries, isNewEntry, workingFile, _ = fs.GetDocumentOrEntry(workingFile)
	}

	fullText := ""
	for i, entry := range entries {
		fullText += fmt.Sprintf("%s %s\n%s\n\n%s\n\n", JOURNAL_DELIMITER, entry.Entry, entry.Timestamp, strings.TrimSpace(entry.Text))
		if Summarize {
			c := color.New(color.FgCyan)
			if i == 0 {
				c.Println("\nSummary:")
			}
			truncated := strings.Fields(entry.Text)
			if len(truncated) > 10 {
				truncated = truncated[:10]
			}
			c.Printf("%10s", strings.Split(entry.Timestamp, " ")[0])
			c = color.New(color.FgHiRed)
			c.Printf(" (%s)", entry.Entry)
			fmt.Printf(" %s\n", strings.Join(truncated, " "))
		}
	}
	if isNewEntry {
		fullText += fmt.Sprintf("%s %s\n%s\n\n\n%s", JOURNAL_DELIMITER, utils.GetRandomMD5Hash(), utils.GetCurrentDate(), "")
	}
	if Summarize {
		fmt.Println("")
		os.Exit(-1)
	}

	// Determine editor
	editor := "vim"
	if utils.Exists(path.Join(homePath, ".config", "bol", "editor")) {
		editorBytes, _ := ioutil.ReadFile(path.Join(homePath, ".config", "bol", "editor"))
		editor = string(editorBytes)
	}
	newText := WriteEntry(fullText, editor, len(entries) == 1)
	if newText == "" {
		return
	}
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

func WriteEntry(text string, editor string, singleEntry bool) string {
	logger.Debug("Editing file")

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
	normal zt
endfu
com! WPCLI call WordProcessorModeCLI()`
		if singleEntry {
			vimrc = `set nocompatible
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
		normal zt
	endfu
	com! WPCLI call WordProcessorModeCLI()`
		}

		err := ioutil.WriteFile(path.Join(ssed.PathToTempFolder, ".vimrc"), []byte(vimrc), 0644)
		if err != nil {
			log.Fatal(err)
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

	// Write the file to load
	ioutil.WriteFile(path.Join(ssed.PathToTempFolder, "temp"), []byte(text), 0644)

	// Try to execute from the same folder
	programPath, _ := osext.ExecutableFolder()
	editor = filepath.Base(editor)
	logger.Debug("Using editor in program path: %s", path.Join(programPath, editor+extension))
	cmd := exec.Command(path.Join(programPath, editor+extension), cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		logger.Debug("Failed using editor in program path: %s", err.Error())
		// Try to execute from system path
		logger.Debug("Using editor in system path: %s", editor+extension)
		cmd := exec.Command(editor+extension, cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err2 := cmd.Run()
		if err2 != nil {
			logger.Debug("Failed using from system path")
			c := color.New(color.FgHiRed)
			c.Printf("\n%s not found in system path or local path, \ndo you have it installed?\n", editor)
			c.Println("\nMake sure you have a editor installed \nin the system or current directory.")
			c.Println("\nSupported editors are:")
			c.Println("- vim:   ftp://ftp.vim.org/pub/vim/pc/vim80-069w32.zip")
			c.Println("- micro: https://github.com/zyedidia/micro/releases/latest")
			c.Println("- emacs")
			c.Println("- nano")
			c.Println("\nYou can switch editors with\n\n\tbol --editor [vim|emacs|micro]")
			fmt.Println("")
			os.Exit(-1)
		}
	}
	fileContents, _ := ioutil.ReadFile(path.Join(ssed.PathToTempFolder, "temp"))
	return strings.TrimSpace(string(fileContents))
}
