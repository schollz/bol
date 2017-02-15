package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/bol/ssed"
	"github.com/urfave/cli"
)

var (
	Version, BuildTime, Build, OS, LastCommit, Editor string
	Debug, Summarize                                  bool
	DontEncrypt, Clean                                bool
	ResetConfig, DumpFile                             bool
	ImportOldFile, ImportFile                         bool
)

func main() {
	// Delete temp files upon exit
	defer ssed.CleanUp()

	// Handle Ctl+C for cleanUp
	// from http://stackoverflow.com/questions/11268943/golang-is-it-possible-to-capture-a-ctrlc-signal-and-run-a-cleanup-function-in
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		ssed.CleanUp()
		os.Exit(1)
	}()

	// App information
	setBuild()
	app := cli.NewApp()
	app.Name = "bol"
	app.Version = Version + " " + Build + " " + BuildTime + " " + OS
	app.Usage = `bol is for synchronized editing of encrypted stuff

	 https://github.com/schollz/bol

EXAMPLE USAGE:
   bol new.txt # create new / edit a document, 'new.txt'
   bol Entry123 # edit a entry, 'Entry123'`

	app.Action = func(c *cli.Context) error {
		// Set the log level
		if Debug {
			ssed.DebugMode()
			DebugMode()
			logger.Debug("Turning on Debug mode")
		}

		if len(Editor) > 0 {
			Editor = strings.TrimSpace(strings.ToLower(Editor))
			if Editor == "vim" || Editor == "emacs" || Editor == "micro" {
				ioutil.WriteFile(path.Join(homePath, ".config", "bol", "editor"), []byte(Editor), 0644)
				fmt.Printf("Editor set to ")
				c := color.New(color.FgHiCyan)
				c.Println(Editor)
			} else {
				c := color.New(color.FgHiRed)
				c.Print(Editor)
				fmt.Println(" is not supported, sorry.")
				fmt.Println("\nSupported editors are:")
				fmt.Println("- vim:   ftp://ftp.vim.org/pub/vim/pc/vim80-069w32.zip")
				fmt.Println("- micro: https://github.com/zyedidia/micro/releases/latest")
				fmt.Println("- emacs")
			}
			return nil
		}

		if Clean {
			ssed.EraseAll()
			fmt.Println("All bol files cleared")
		} else {
			workingFile := c.Args().Get(0)
			Run(workingFile, ResetConfig, DumpFile)
		}
		return nil
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "Turn on debug mode",
			Destination: &Debug,
		},
		cli.BoolFlag{
			Name:        "clean",
			Usage:       "Deletes all bol files",
			Destination: &Clean,
		},
		cli.BoolFlag{
			Name:        "config",
			Usage:       "Configure",
			Destination: &ResetConfig,
		},
		cli.BoolFlag{
			Name:        "dump",
			Usage:       "Dump the current documents",
			Destination: &DumpFile,
		},
		cli.StringFlag{
			Name:        "editor",
			Usage:       "express which `editor` to use (micro / vim / emacs)",
			Destination: &Editor,
		},
		// cli.BoolFlag{
		// 	Name:        "importold",
		// 	Usage:       "Import `document` (JRNL-format)",
		// 	Destination: &bol.ImportOldFlag,
		// },
		// cli.BoolFlag{
		// 	Name:        "import",
		// 	Usage:       "Import `document`",
		// 	Destination: &bol.ImportFlag,
		// },
		// cli.BoolFlag{
		// 	Name:        "export",
		// 	Usage:       "Export `document`",
		// 	Destination: &bol.Export,
		// },
		// cli.BoolFlag{
		// 	Name:        "all, a",
		// 	Usage:       "Edit all of the document",
		// 	Destination: &bol.All,
		// },
		// cli.BoolFlag{
		// 	Name:        "delete",
		// 	Usage:       "Delete `X`, where X is a document or entry",
		// 	Destination: &bol.DeleteFlag,
		// },
		cli.BoolFlag{
			Name:        "summary",
			Usage:       "Gets summary",
			Destination: &Summarize,
		},
		// cli.BoolFlag{
		// 	Name:        "stats",
		// 	Usage:       "Print stats",
		// 	Destination: &bol.ShowStats,
		// },
	}
	app.Run(os.Args)
}

func setBuild() {
	if len(Build) == 0 {
		cwd, _ := os.Getwd()
		defer os.Chdir(cwd)
		Build = "dev"
		Version = "dev"
		BuildTime = time.Now().String()
		err := os.Chdir(path.Join(os.Getenv("GOPATH"), "src", "github.com", "schollz", "bol"))
		if err != nil {
			return
		}
		cmd := exec.Command("git", "log", "-1", "--pretty=format:'%h||%ad'")
		stdout, err := cmd.Output()
		if err != nil {
			return
		}
		items := strings.Split(string(stdout), "||")
		LastCommit = strings.Replace(items[1], "'", "", -1)
		Build = strings.Replace(items[0], "'", "", -1)
		BuildTime = LastCommit
	} else {
		Build = Build[0:7]
	}
}
