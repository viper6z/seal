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

func TestGetDiskBlobs(t *testing.T) {
	blobs, err := getDiskBlobs("/home/granl/seal")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(blobs)
}

func TestCompareBlobs(t *testing.T) {
	verdict, err := compareBlobs("/home/granl/seal", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(verdict)
}
