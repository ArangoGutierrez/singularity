// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"text/template"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/auth"
	"github.com/spf13/cobra"
)

// Global variables for RunSy CLI
var (
	debug      bool
	silent     bool
	verbose    bool
	quiet      bool
	bundlePath string
)

var (
	// TokenFile holds the path to the sylabs auth token file
	defaultTokenFile, tokenFile string
	// authToken holds the sylabs auth token
	authToken, authWarning string
)

func init() {
	RunsyCmd.Flags().SetInterspersed(false)
	RunsyCmd.PersistentFlags().SetInterspersed(false)

	templateFuncs := template.FuncMap{
		"TraverseParentsUses": TraverseParentsUses,
	}
	cobra.AddTemplateFuncs(templateFuncs)

	RunsyCmd.SetHelpTemplate(docs.HelpTemplate)
	RunsyCmd.SetUsageTemplate(docs.UseTemplate)

	RunsyCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Print debugging information")
	RunsyCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Only print errors")
	RunsyCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all normal output")
	RunsyCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Increase verbosity +1")
	usr, err := user.Current()
	if err != nil {
		sylog.Fatalf("Couldn't determine user home directory: %v", err)
	}
	defaultTokenFile = path.Join(usr.HomeDir, ".singularity", "sylabs-token")

	RunsyCmd.Flags().StringVar(&tokenFile, "tokenfile", defaultTokenFile, "path to the file holding your sylabs authentication token")
	VersionCmd.Flags().SetInterspersed(false)
	RunsyCmd.AddCommand(VersionCmd)
}

// RunsyCmd is the base command when called without any subcommands
var RunsyCmd = &cobra.Command{
	TraverseChildren:      true,
	DisableFlagsInUseLine: true,
	Run: nil,

	Use:     docs.RunsyUse,
	Version: fmt.Sprintf("%v-%v\n", buildcfg.PACKAGE_VERSION, buildcfg.GIT_VERSION),
	Short:   docs.RunsyShort,
	Long:    docs.RunsyLong,
	Example: docs.RunsyExample,
}

// ExecuteRunsyCmd adds all child commands to the root command and sets
// flags appropriately. This is called by main.main(). It only needs to happen
// once to the root command (singularity).
func ExecuteRunsyCmd() {
	if err := RunsyCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// TraverseParentsUses walks the parent commands and outputs a properly formatted use string
func TraverseParentsUses(cmd *cobra.Command) string {
	if cmd.HasParent() {
		return TraverseParentsUses(cmd.Parent()) + cmd.Use + " "
	}

	return cmd.Use + " "
}

// VersionCmd displays installed singularity version
var VersionCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%v-%v\n", buildcfg.PACKAGE_VERSION, buildcfg.GIT_VERSION)
	},

	Use:   "version",
	Short: "Show application version",
}

// sylabsToken process the authentication Token
// priority default_file < env < file_flag
func sylabsToken(cmd *cobra.Command, args []string) {
	if val := os.Getenv("SYLABS_TOKEN"); val != "" {
		authToken = val
	}
	if tokenFile != defaultTokenFile {
		authToken, authWarning = auth.ReadToken(tokenFile)
	}
	if authToken == "" {
		authToken, authWarning = auth.ReadToken(defaultTokenFile)
	}
	if authToken == "" && authWarning == auth.WarningTokenFileNotFound {
		sylog.Warningf("%v : Only pulls of public images will succeed", authWarning)
	}
}