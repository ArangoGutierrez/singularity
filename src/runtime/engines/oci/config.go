// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
)

const (
	// Name is the name of the runtime.
	Name = "oci"
	// StatePath holds the path for writing the state files
	// on creation runsy will write a stat file into StatePath/<container-ID>/state.json
	StatePath = "/var/run/runsy"
)

// EngineConfig holds the config info required to create/start/stop
// the oci-singularity engine
type EngineConfig struct {
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

// MarshalJSON is for json.Marshaler
func (e *EngineConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalJSON is for json.Unmarshaler
func (e *EngineConfig) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, e)
}
