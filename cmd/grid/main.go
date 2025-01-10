package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andrewpmartinez/grid/dump"
	"github.com/andrewpmartinez/grid/gui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{}
var logger = logrus.New()

func init() {
	guiCmd := &cobra.Command{
		Use:  "gui <file>",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := args[0]

			if path == "version" {
				return
			}

			absPath, err := filepath.Abs(path)

			if err != nil {
				logger.Errorf("invalid path, could not transform to absolute path: %s", path)
				os.Exit(1)
			}

			_, err = os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					logger.Errorf("invalid file, does not exist: %s", absPath)
				} else {
					logger.Errorf("unexpected error attempting to check file %s: %v", absPath, err)
				}
				os.Exit(1)
			}

			win := gui.NewDumpWindow(logger)
			win.LoadFile(absPath)
			win.Run()
		},
	}

	parseCmd := &cobra.Command{
		Use:  "parse <file>",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			absPath, err := filepath.Abs(args[0])
			if err != nil {
				logger.Errorf("could not find the absolute path for file [%s]", args[0])
				os.Exit(1)
			}

			outDump, err := dump.ParseFile(absPath, logger)

			if err != nil {
				logger.Errorf("error parsing file [%s]: %v", absPath, err)
				os.Exit(1)
			}

			logger.Infof("parsed %d go routines", len(outDump.Routines))
		},
	}

	versionCmd := &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("version: %s\n"+
				"commit: %s\n"+
				"branch: %s\n"+
				"build date: %s\n",
				dump.Version, dump.Commit, dump.Branch, dump.BuildDate)
		},
	}

	rootCmd.AddCommand(guiCmd)
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error(err)
	}
}
