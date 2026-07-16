package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const (
	originalCompose  = "original compose"
	candidateCompose = "candidate compose"
)

func TestDeploymentTransactionSuccess(t *testing.T) {
	candidatePath, application := setupDeploymentFixture(t, true)

	err := publishDeployment(candidatePath, application)

	if err != nil {
		t.Fatalf("publishDeployment returned an unexpected error: %v", err)
	}

	assertFileContent(t, "compose.yaml", candidateCompose)
	assertPathDoesNotExist(t, "backup.yaml")
	assertPathDoesNotExist(t, candidatePath)

	nginxPath := filepath.Join(
		"nginx",
		"conf.d",
		"generated",
		"new.conf",
	)

	info, err := os.Stat(nginxPath)
	if err != nil {
		t.Fatalf("expected generated Nginx file %q to exist: %v", nginxPath, err)
	}
	if info.IsDir() {
		t.Fatalf("expected %q to be a file, but it is a directory", nginxPath)
	}
}

func TestDeploymentRestoresComposeWhenNginxTempCreationFails(t *testing.T) {
	candidatePath, application := setupDeploymentFixture(t, false)

	err := publishDeployment(candidatePath, application)

	if err == nil {
		t.Fatal("publishDeployment returned nil, want an error")
	}

	assertFileContent(t, "compose.yaml", originalCompose)
	assertPathDoesNotExist(t, "backup.yaml")
	assertPathDoesNotExist(t, candidatePath)

	nginxPath := filepath.Join(
		"nginx",
		"conf.d",
		"generated",
		"new.conf",
	)
	assertPathDoesNotExist(t, nginxPath)
}

func TestDeploymentRestoresComposeWhenNginxPublishFails(t *testing.T) {
	candidatePath, application := setupDeploymentFixture(t, true)

	generatedDirectory := filepath.Join(
		"nginx",
		"conf.d",
		"generated",
	)
	finalPath := filepath.Join(generatedDirectory, "new.conf")

	if err := os.Mkdir(finalPath, 0o755); err != nil {
		t.Fatalf("create blocking Nginx destination directory: %v", err)
	}

	err := publishDeployment(candidatePath, application)

	if err == nil {
		t.Fatal("publishDeployment returned nil, want an error")
	}

	assertFileContent(t, "compose.yaml", originalCompose)
	assertPathDoesNotExist(t, "backup.yaml")
	assertPathDoesNotExist(t, candidatePath)

	entries, err := os.ReadDir(generatedDirectory)
	if err != nil {
		t.Fatalf("read generated Nginx directory: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf(
			"generated Nginx directory contains %d entries, want only new.conf",
			len(entries),
		)
	}

	if entries[0].Name() != "new.conf" || !entries[0].IsDir() {
		t.Fatalf(
			"unexpected path remains after replaceNginxConf failure: %q",
			entries[0].Name(),
		)
	}
}

func TestReplaceComposeRestoresOriginalWhenCandidatePublishFails(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	writeFixtureFile(t, "compose.yaml", originalCompose)

	candidatePath := "missing-candidate.yaml"

	err := replaceCompose(candidatePath)

	if err == nil {
		t.Fatal("replaceCompose returned nil, want an error")
	}

	assertFileContent(t, "compose.yaml", originalCompose)
	assertPathDoesNotExist(t, "backup.yaml")
	assertPathDoesNotExist(t, candidatePath)
}

func TestReplaceComposeRejectsStaleBackup(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	writeFixtureFile(t, "compose.yaml", originalCompose)
	writeFixtureFile(t, "candidate.yaml", candidateCompose)
	writeFixtureFile(t, "backup.yaml", "stale backup")

	err := replaceCompose("candidate.yaml")

	if err == nil {
		t.Fatal("replaceCompose returned nil, want an error")
	}

	assertFileContent(t, "compose.yaml", originalCompose)
	assertFileContent(t, "backup.yaml", "stale backup")
	assertFileContent(t, "candidate.yaml", candidateCompose)
}

func setupDeploymentFixture(
	t *testing.T,
	createGeneratedDirectory bool,
) (string, Application) {
	t.Helper()

	root := t.TempDir()
	t.Chdir(root)

	writeFixtureFile(t, "compose.yaml", originalCompose)

	candidatePath := "candidate.yaml"
	writeFixtureFile(t, candidatePath, candidateCompose)

	if createGeneratedDirectory {
		generatedDirectory := filepath.Join(
			"nginx",
			"conf.d",
			"generated",
		)

		if err := os.MkdirAll(generatedDirectory, 0o755); err != nil {
			t.Fatalf("create generated Nginx directory: %v", err)
		}
	}

	application := Application{
		Name:                "new",
		InternalPort:        8080,
		ExposureType:        "public",
		AllowedPublicRoutes: []string{"/"},
	}

	return candidatePath, application
}

func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file %q: %v", path, err)
	}
}

func readFixtureFile(t *testing.T, path string) []byte {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture file %q: %v", path, err)
	}

	return content
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()

	actual := readFixtureFile(t, path)

	if !bytes.Equal(actual, []byte(expected)) {
		t.Fatalf(
			"file %q contains %q, want %q",
			path,
			actual,
			expected,
		)
	}
}

func assertPathDoesNotExist(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("expected path %q not to exist", path)
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inspect path %q: %v", path, err)
	}
}
