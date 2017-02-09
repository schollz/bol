package main

import (
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/schollz/bol/ssed"
	"github.com/urfave/cli"
)

var (
	Version, BuildTime, Build, OS, LastCommit string
	Debug                                     bool
	DontEncrypt, Clean                        bool
	ResetConfig, DumpFile                     bool
	ImportOldFile, ImportFile                 bool
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
	app.Usage = `bol is for distributed editing of encrypted stuff

	 https://github.com/schollz/bol

EXAMPLE USAGE:
   bol new.txt # create new / edit a document, 'new.txt'
   bol Entry123 # edit a entry, 'Entry123'
   bol --summary # list a summary of all entries
   bol --search "dogs cats" # find entries that mention 'dogs' or 'cats'`

	app.Action = func(c *cli.Context) error {
		// Set the log level
		if Debug {
			ssed.DebugMode()
			DebugMode()
		}

		if Clean {
			ssed.EraseAll()
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
		// cli.BoolFlag{
		// 	Name:        "summary",
		// 	Usage:       "Gets summary",
		// 	Destination: &bol.Summarize,
		// },
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
