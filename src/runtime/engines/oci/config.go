// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"path/filepath"
)

const (
	// Name is the name of the runtime.
	Name = "oci"
	// StatePath holds the path for writing the state files
	// on creation runsy will write a stat file into StatePath/<container-ID>/state.json
	StatePath = "/var/run/runsy"
)

// JSONConfig stores engine specific confguration that is allowed to be set by the user
type JSONConfig struct {
	Image            string `json:"image"`
	WritableImage    bool   `json:"writableImage,omitempty"`
	OverlayImage     string `json:"overlayImage,omitempty"`
	OverlayFsEnabled bool   `json:"overlayFsEnabled,omitempty"`
	Contain          bool   `json:"container,omitempty"`
	Nv               bool   `json:"nv,omitempty"`
	IsInstance       bool   `json:"isInstance,omitempty"`
	RunPrivileged    bool   `json:"runPrivileged,omitempty"`
	AddCaps          string `json:"addCaps,omitempty"`
	DropCaps         string `json:"dropCaps,omitempty"`
	AllowSUID        bool   `json:"allowSUID,omitempty"`
	KeepPrivs        bool   `json:"keepPrivs,omitempty"`
	NoPrivs          bool   `json:"noPrivs,omitempty"`
	Home             string `json:"home,omitempty"`
}

// EngineConfig holds the config info required to create/start/stop
// the oci-singularity engine
type EngineConfig struct {
	JSON *JSONConfig `json:"jsonConfig"`
}

// MarshalJSON is for json.Marshaler
func (e *EngineConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.JSON)
}

// UnmarshalJSON is for json.Unmarshaler
func (e *EngineConfig) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, e.JSON)
}

// SetImage sets the container image path to be used by EngineConfig.JSON.
func (e *EngineConfig) SetImage(name string) {
	abs, _ := filepath.Abs(name)
	e.JSON.Image = abs
}

// SetInstance sets if container run as instance or not.
func (e *EngineConfig) SetInstance(instance bool) {
	e.JSON.IsInstance = instance
}
