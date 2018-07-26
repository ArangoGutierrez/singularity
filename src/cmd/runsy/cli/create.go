// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"encoding/json"
	"os"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/buildcfg"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/pkg/util/exec"
	util "github.com/singularityware/singularity/src/pkg/util/oci"
	"github.com/singularityware/singularity/src/runtime/engines/common/config"
	common "github.com/singularityware/singularity/src/runtime/engines/common/oci"
	"github.com/singularityware/singularity/src/runtime/engines/oci"

	"github.com/spf13/cobra"
)

func init() {
	CreateCmd.Flags().SetInterspersed(false)

	cwd, err := os.Getwd()
	if err != nil {
		sylog.Fatalf("%v", err)
	}

	CreateCmd.Flags().StringVarP(&bundlePath, "bundle", "b", cwd, "path to singularity image file (SIF), default to current directory")

	RunsyCmd.AddCommand(CreateCmd)

}

// CreateCmd runsy create cmd
var CreateCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// WIP --->
		lvl := "0"
		// set vars from cli
		cID := args[0]
		sifPath := args[1]

		if verbose {
			lvl = "2"
		}
		if debug {
			lvl = "5"
		}

		wrapper := buildcfg.SBINDIR + "/wrapper"

		engineConfig := &oci.EngineConfig{
			Image:      sifPath,
			IsInstance: true,
		}

		// read OCI runtime spec on SIF bundle
		spec, err := util.LoadConfigSpec(sifPath)
		if err != nil {
			sylog.Fatalf("Couldn't load config spec from SIF:\t%s", err)
		}

		ociConfig := &common.Config{Spec: *spec}
		ociConfig.Generator = generate.NewFromSpec(&ociConfig.Spec)
		ociConfig.Generator.SetProcessArgs(spec.Process.Args)

		Env := []string{"SINGULARITY_MESSAGELEVEL=" + lvl, "SRUNTIME=singularity"}
		progname := "Sylabs oci runtime"

		cfg := &config.Common{
			EngineName:   oci.Name,
			ContainerID:  cID,
			OciConfig:    ociConfig,
			EngineConfig: engineConfig,
		}

		configData, err := json.Marshal(cfg)
		if err != nil {
			sylog.Fatalf("CLI Failed to marshal CommonEngineConfig: %s\n", err)
		}

		if err := exec.Pipe(wrapper, []string{progname}, Env, configData); err != nil {
			sylog.Fatalf("%s", err)
		}
		// <---
	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsyCreateUse,
	Short: docs.RunsyCreateShort,
	Long:  docs.RunsyCreateLong,
}
