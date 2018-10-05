// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/sylabs/singularity/src/pkg/test"
)

func TestHelpSingularity(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{"NoCommand", []string{}},
		{"FlagShort", []string{"-h"}},
		{"FlagLong", []string{"--help"}},
		{"Command", []string{"help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithOutPrivilege(func(t *testing.T) {
			cmd := exec.Command(cmdPath, tt.argv...)
			if b, err := cmd.CombinedOutput(); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			}
		}, tt.name))
	}
}

func TestHelpFailure(t *testing.T) {
	if !*runDisabled {
		t.Skip("disabled until issue addressed") // TODO
	}

	tests := []struct {
		name string
		argv []string
	}{
		{"HelpBogus", []string{"help", "bogus"}},
		{"BogusHelp", []string{"bogus", "help"}},
		{"HelpInstanceBogus", []string{"help", "instance", "bogus"}},
		{"ImageBogusHelp", []string{"image", "bogus", "help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithOutPrivilege(func(t *testing.T) {
			cmd := exec.Command(cmdPath, tt.argv...)
			if b, err := cmd.CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}, tt.name))
	}
}

func TestHelpCommands(t *testing.T) {
	cmds := []struct {
		name string
		argv []string
	}{
		{"Apps", []string{"apps"}},
		{"Bootstrap", []string{"bootstrap"}},
		{"Build", []string{"build"}},
		{"Check", []string{"check"}},
		{"Create", []string{"create"}},
		{"Exec", []string{"exec"}},
		{"Inspect", []string{"inspect"}},
		{"Mount", []string{"mount"}},
		{"Pull", []string{"pull"}},
		{"Run", []string{"run"}},
		{"Shell", []string{"shell"}},
		{"Test", []string{"test"}},
		{"InstanceDotStart", []string{"instance.start"}},
		{"InstanceDotList", []string{"instance.list"}},
		{"InstanceDotStop", []string{"instance.stop"}},
		{"InstanceStart", []string{"instance", "start"}},
		{"InstanceList", []string{"instance", "list"}},
		{"InstanceStop", []string{"instance", "stop"}},
	}

	for _, tt := range cmds {
		t.Run(tt.name, test.WithOutPrivilege(func(t *testing.T) {
			tests := []struct {
				name string
				argv []string
				skip bool
			}{
				{"PostFlagShort", append(tt.argv, "-h"), true}, // TODO
				{"PostFlagLong", append(tt.argv, "--help"), false},
				{"PostCommand", append(tt.argv, "help"), false},
				{"PreFlagShort", append([]string{"-h"}, tt.argv...), false},
				{"PreFlagLong", append([]string{"--help"}, tt.argv...), false},
				{"PreCommand", append([]string{"help"}, tt.argv...), false},
			}
			for _, ts := range tests {
				if ts.skip && !*runDisabled {
					t.Skip("disabled until issue addressed")
				}

				t.Run(ts.name, test.WithOutPrivilege(func(t *testing.T) {
					cmd := exec.Command(cmdPath, tt.argv...)
					if b, err := cmd.CombinedOutput(); err != nil {
						t.Log(string(b))
						t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
					}
				}, path.Join(tt.name, ts.name)))
			}
		}, tt.name))
	}
}
