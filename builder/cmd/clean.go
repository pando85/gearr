package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"transcoder/helper/command"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "c",
	Long:  `Clean Environment`,
	Run: func(cmd *cobra.Command, args []string) {
		buildDir := filepath.Join(command.GetWD(),"build")
		if err:=os.RemoveAll(buildDir);err!=nil {
			panic(err)
		}
	},
}


func init() {
	RootCmd.AddCommand(cleanCmd)
}
