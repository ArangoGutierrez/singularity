/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/singularityware/singularity/src/pkg/sylog"
	"gopkg.in/mgo.v2/bson"
)

// BuildAndPush contains the build request and push request
type BuildAndPush struct {
	Client     http.Client
	LibraryRef string
	Definition Definition
	HTTPAddr   string
	AuthHeader string
}

// NewBuildAndPush creates a RemoteBuilder with the specified details.
func NewBuildAndPush(libraryRef string, d Definition, httpAddr, authToken string) (bp *BuildAndPush, err error) {
	if !isLibraryPushRef(libraryRef) {
		return fmt.Errorf("Not a valid library reference: %s", libraryRef)
	}

	bp = &BuildAndPush{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		LibraryRef: imaglibraryRefePath,
		Definition: d,
		HTTPAddr:   httpAddr,
	}
	if authToken != "" {
		bp.AuthHeader = fmt.Sprintf("Bearer %s", authToken)
	}

	return bp, nil
}

// Build is responsible for making the request via the REST API to the remote builder
func (bp *BuildAndPush) Build(ctx context.Context) (err error) {
	// Send build request to Remote Build Service
	rd, err := bp.doBuildRequest(ctx)
	if err != nil {
		err = errors.Wrap(err, "failed to post request to remote build service")
		sylog.Warningf("%v\n", err)
		return
	}

	rd, err = bp.doStatusRequest(ctx, rd.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get status from remote build service")
		sylog.Warningf("%v\n", err)
		return
	}

	fmt.Printf("Build submited with ID: %v", rd.ID)

	return
}

// doBuild&Push Request creates a new build on a Remote Build Service
// that will push the image to the library afterwards
func (bp *BuildAndPush) doBuildRequest(ctx context.Context) (rd ResponseData, err error) {
	b, err := json.Marshal(RequestData{
		Definition: bp.Definition,
		LibraryRef: bp.LibraryRef,
		IsDetached: true,
	})
	if err != nil {
		return
	}

	url := url.URL{Scheme: "http", Host: bp.HTTPAddr, Path: "/v1/buildandpush"}
	req, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(b))
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	if bp.AuthHeader != "" {
		req.Header.Set("Authorization", bp.AuthHeader)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := bp.Client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		err = errors.New(res.Status)
		return
	}

	err = json.NewDecoder(res.Body).Decode(&rd)
	return
}

// doStatusRequest gets the status of a build from the Remote Build Service
func (bp *BuildAndPush) doStatusRequest(ctx context.Context, id bson.ObjectId) (rd ResponseData, err error) {
	url := url.URL{Scheme: "http", Host: rb.HTTPAddr, Path: "/v1/build/" + id.Hex()}
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	if rb.AuthHeader != "" {
		req.Header.Set("Authorization", rb.AuthHeader)
	}

	res, err := rb.Client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return
	}

	err = json.NewDecoder(res.Body).Decode(&rd)
	return
}

func isLibraryPushRef(libraryRef string) bool {
	// For push we allow specifying multiple tags, delimited with ,
	match, _ := regexp.MatchString("^(library://)?([a-z0-9]+(?:[._-][a-z0-9]+)*/){2}([a-z0-9]+(?:[._-][a-z0-9]+)*)(:[a-z0-9]+(?:[,._-][a-z0-9]+)*)?$", libraryRef)
	return match
}
