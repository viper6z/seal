package main

import (
	"testing"
)

func TestGetTargetBlobs(t *testing.T) {
	blobs, err := getTargetBlobs("/home/granl/seal", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(blobs)
}
