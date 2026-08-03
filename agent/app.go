package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: agenttest <repository-path>")
		os.Exit(2)
	}

	repoPath := os.Args[1]
	stagingPath := "/tmp/staging"
	statePath := filepath.Join(repoPath, ".seal", "applied")

	applied, err := loadAppliedCommit(statePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load applied commit:", err)
		os.Exit(1)
	}
	// if applied is empty firstRun = true
	firstRun := applied == ""
	target, err := fetchTargetCommit(repoPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch target commit:", err)
		os.Exit(1)
	}

	fmt.Println("applied commit:", applied)
	fmt.Println("target commit:", target)
	// if its not the first run and theres no delta between applied and target we return early
	if !firstRun && target == applied {
		fmt.Println("already reconciled")
		return
	}

	changed := true

	if !firstRun {
		//check if the diff touches our managed paths (nginx and compose)
		changed, err = managedPathsChanged(repoPath, target, applied)
		if err != nil {
			fmt.Fprintln(os.Stderr, "compare managed paths:", err)
			os.Exit(1)
		}
	}

	if !changed {
		//if nothing is changed in the managed paths we just save the state and then return early
		if err := saveAppliedCommit(statePath, target); err != nil {
			fmt.Fprintln(os.Stderr, "advance applied commit:", err)
			os.Exit(1)
		}

		fmt.Println("target contains no managed configuration changes")
		return
	}

	if err := reconcileConfigs(repoPath, target, stagingPath); err != nil {
		fmt.Fprintln(os.Stderr, "stage target configuration:", err)
		os.Exit(1)
	}
	//at this point our staging path will contain exactly a compose.yaml and a nginx/conf.d/ dir. this makes it easy to validate.
	fmt.Println("staged configuration at:", stagingPath)

	if err := validateStaged(stagingPath); err != nil {
		fmt.Fprintln(os.Stderr, "validate staged configuration:", err)
		os.Exit(1)
	}

	fmt.Println("staged configuration is valid")

	backupPath, err := publishStagedConfigs(repoPath, stagingPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "publish staged configuration:", err)
		os.Exit(1)
	}

	fmt.Println("staged configuration published")
	fmt.Println("previous configuration backed up at:", backupPath)

	if err := applyRuntime(repoPath); err != nil {
		applyErr := err
		//if applying the runtime returns an error we call rollbackPublishedConfigs
		if restoreErr := rollbackPublishedConfigs(repoPath, backupPath); restoreErr != nil {
			fmt.Fprintln(
				os.Stderr,
				errors.Join(
					fmt.Errorf("apply new runtime: %w", applyErr),
					fmt.Errorf("restore previous files: %w", restoreErr),
				),
			)
			os.Exit(1)
		}
		//after the backup has been restored we need to reapply it to the runtime
		if restoreRuntimeErr := applyRuntime(repoPath); restoreRuntimeErr != nil {
			fmt.Fprintln(
				os.Stderr,
				errors.Join(
					fmt.Errorf("apply new runtime: %w", applyErr),
					fmt.Errorf("reapply previous runtime: %w", restoreRuntimeErr),
				),
			)
			os.Exit(1)
		}

		fmt.Fprintln(
			os.Stderr,
			"runtime apply failed; previous configuration restored",
		)
		os.Exit(1)
	}
	//save the commit sha of the succeeded apply
	fmt.Println("runtime configuration applied")
	if err := saveAppliedCommit(statePath, target); err != nil {
		fmt.Fprintln(os.Stderr, "save applied commit:", err)
		fmt.Fprintln(os.Stderr, "backup retained at:", backupPath)
		os.Exit(1)
	}

	fmt.Println("applied commit recorded:", target)
	if err := os.RemoveAll(backupPath); err != nil {
		fmt.Fprintln(
			os.Stderr,
			"warning: runtime succeeded but backup cleanup failed:",
			err,
		)
	} else {
		fmt.Println("configuration backup removed")
	}
}

// fetches origin/main and returns its current commit SHA.
func fetchTargetCommit(repoPath string) (target string, err error) {
	//fetch main
	cmd := exec.Command("git", "fetch", "origin", "main")
	cmd.Dir = repoPath
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	//convert to SHA-1 hash and return as target
	cmd = exec.Command("git", "rev-parse", "--verify", "origin/main")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func managedPathsChanged(repoPath string, target string, applied string) (diff bool, err error) {
	cmd := exec.Command("git", "diff", "--quiet", applied, target, "--", "compose.yaml", "nginx/conf.d/")
	cmd.Dir = repoPath
	err = cmd.Run()
	if err == nil {
		return false, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, err
}

func reconcileConfigs(repoPath, target, stagingPath string) error {
	//ensure staging path is empty by removing it and recreating it
	if err := os.RemoveAll(stagingPath); err != nil {
		return fmt.Errorf("remove staging directory: %w", err)
	}

	if err := os.MkdirAll(stagingPath, 0o755); err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	// create temporary tar file in temporary dir just to guarantee git can write to a unique filename
	archiveFile, err := os.CreateTemp("", "seal-configs-*.tar")
	if err != nil {
		return fmt.Errorf("create temporary archive: %w", err)
	}

	archivePath := archiveFile.Name()
	//close it insantly because only git will touch it
	if err := archiveFile.Close(); err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("close temporary archive: %w", err)
	}
	//defer to remove it in case the git operation goes south
	defer os.Remove(archivePath)
	//run git archive command writing the files as they are in the target commit to our tar file
	archiveCmd := exec.Command(
		"git",
		"archive",
		"--format=tar",
		"--output="+archivePath,
		target,
		"compose.yaml",
		"nginx/conf.d/",
	)
	archiveCmd.Dir = repoPath

	if output, err := archiveCmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"archive target configuration: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}
	// copy the files in the tarball to our stagingPath
	tarCmd := exec.Command(
		"tar",
		"-xf",
		archivePath,
		"-C",
		stagingPath,
	)

	if output, err := tarCmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"extract target configuration: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return nil
}

// we validate compose config and also start a temporary nginx container to validate
func validateStaged(stagingPath string) error {
	composePath := filepath.Join(stagingPath, "compose.yaml")
	//validate compose config
	composeCmd := exec.Command(
		"docker",
		"compose",
		"-f",
		composePath,
		"config",
		"--quiet",
	)

	if output, err := composeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"validate staged compose: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}
	//list all services
	servicesCmd := exec.Command(
		"docker",
		"compose",
		"-f",
		composePath,
		"config",
		"--services",
	)
	//capture services in output
	output, err := servicesCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"list staged compose services: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}
	//make a slice containing our service names
	services := strings.Fields(string(output))

	//build a docker run command for nginx where we pass add-host with loopback address for each service so they resolve on the validation container
	nginxArgs := []string{
		"run",
		"--rm",
	}
	// this is where we add a -add-host entry for each service in the slice, bound to loopback address
	for _, service := range services {
		nginxArgs = append(
			nginxArgs,
			"--add-host",
			service+"=127.0.0.1",
		)
	}

	nginxArgs = append(
		nginxArgs,
		"--mount",
		"type=bind,src="+filepath.Join(stagingPath, "nginx", "conf.d")+
			",dst=/etc/nginx/conf.d,readonly",
		"nginx:1.30.3-alpine",
		"nginx",
		"-t",
	)

	nginxCmd := exec.Command("docker", nginxArgs...)

	output, err = nginxCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"validate staged nginx: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return nil
}
func copyFile(sourcePath, destinationPath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat source file %s: %w", sourcePath, err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", sourcePath, err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return fmt.Errorf("create destination parent: %w", err)
	}

	destination, err := os.OpenFile(
		destinationPath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		sourceInfo.Mode().Perm(),
	)
	if err != nil {
		return fmt.Errorf("create destination file %s: %w", destinationPath, err)
	}

	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		return fmt.Errorf("copy %s to %s: %w", sourcePath, destinationPath, err)
	}

	if err := destination.Close(); err != nil {
		return fmt.Errorf("close destination file %s: %w", destinationPath, err)
	}

	return nil
}

func copyDirectory(sourceRoot, destinationRoot string) error {
	return filepath.WalkDir(
		sourceRoot,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			relativePath, err := filepath.Rel(sourceRoot, path)
			if err != nil {
				return fmt.Errorf("calculate relative path: %w", err)
			}

			destinationPath := filepath.Join(destinationRoot, relativePath)

			if entry.IsDir() {
				info, err := entry.Info()
				if err != nil {
					return fmt.Errorf("inspect directory %s: %w", path, err)
				}

				if err := os.MkdirAll(
					destinationPath,
					info.Mode().Perm(),
				); err != nil {
					return fmt.Errorf(
						"create destination directory %s: %w",
						destinationPath,
						err,
					)
				}

				return nil
			}

			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing to copy symlink: %s", path)
			}

			return copyFile(path, destinationPath)
		},
	)
}

func restoreBackup(
	liveCompose string,
	liveNginx string,
	backupCompose string,
	backupNginx string,
	backupRoot string,
) error {
	var restoreErrors []error

	//remove anything partially published.
	if err := os.Remove(liveCompose); err != nil && !os.IsNotExist(err) {
		restoreErrors = append(
			restoreErrors,
			fmt.Errorf("remove published compose: %w", err),
		)
	}

	if err := os.RemoveAll(liveNginx); err != nil {
		restoreErrors = append(
			restoreErrors,
			fmt.Errorf("remove published nginx directory: %w", err),
		)
	}

	//restore the previous live configuration.
	if err := os.Rename(backupCompose, liveCompose); err != nil {
		restoreErrors = append(
			restoreErrors,
			fmt.Errorf("restore compose backup: %w", err),
		)
	}

	if err := os.Rename(backupNginx, liveNginx); err != nil {
		restoreErrors = append(
			restoreErrors,
			fmt.Errorf("restore nginx backup: %w", err),
		)
	}
	//return the full list of errors, meaning this function does as much as it can
	if len(restoreErrors) > 0 {
		//keep the backup directory for manual recovery.
		return errors.Join(restoreErrors...)
	}

	return os.RemoveAll(backupRoot)
}

func publishStagedConfigs(liveRoot string, stagingPath string) (string, error) {
	//make temporary directory INSIDE the live root that is gonna be our candidate
	candidateRoot, err := os.MkdirTemp(
		liveRoot,
		".seal-candidate-*",
	)

	if err != nil {
		return "", fmt.Errorf("create candidate directory: %w", err)
	}
	defer os.RemoveAll(candidateRoot)
	//make variable path strings for copying over the files
	candidateCompose := filepath.Join(candidateRoot, "compose.yaml")
	candidateNginx := filepath.Join(
		candidateRoot,
		"nginx",
		"conf.d",
	)
	//path strings
	stagedCompose := filepath.Join(stagingPath, "compose.yaml")
	stagedNginx := filepath.Join(
		stagingPath,
		"nginx",
		"conf.d",
	)
	//copy our staged compose into the candidate directory
	if err := copyFile(stagedCompose, candidateCompose); err != nil {
		return "", fmt.Errorf("prepare candidate compose: %w", err)
	}
	//copy staged nginx conf into candidate directory
	if err := copyDirectory(stagedNginx, candidateNginx); err != nil {
		return "", fmt.Errorf("prepare candidate nginx: %w", err)
	}
	//make a temporary backup directory inside our liveRoot
	backupRoot, err := os.MkdirTemp(
		liveRoot,
		".seal-backup-*",
	)
	if err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}
	//string paths for backup stuffb
	backupCompose := filepath.Join(backupRoot, "compose.yaml")
	backupNginx := filepath.Join(
		backupRoot,
		"nginx",
		"conf.d",
	)
	//create paths that will contain backup compose and nginx
	if err := os.MkdirAll(
		filepath.Dir(backupNginx),
		0o755,
	); err != nil {
		os.RemoveAll(backupRoot)
		return "", fmt.Errorf("create backup nginx directory: %w", err)
	}
	//the current paths
	liveCompose := filepath.Join(liveRoot, "compose.yaml")
	liveNginx := filepath.Join(
		liveRoot,
		"nginx",
		"conf.d",
	)

	//Back up the current live configuration.
	if err := os.Rename(liveCompose, backupCompose); err != nil {
		os.RemoveAll(backupRoot)
		return "", fmt.Errorf("back up live compose: %w", err)
	}

	if err := os.Rename(liveNginx, backupNginx); err != nil {
		//if this fails we reverse the rename of the compose
		restoreErr := os.Rename(backupCompose, liveCompose)
		if restoreErr != nil {
			return "", errors.Join(
				fmt.Errorf("back up live nginx: %w", err),
				fmt.Errorf("restore compose backup: %w", restoreErr),
			)
		}

		os.RemoveAll(backupRoot)
		return "", fmt.Errorf("back up live nginx: %w", err)
	}

	//Move candidate to live path
	if err := os.Rename(candidateCompose, liveCompose); err != nil {
		// if it errors, at this point we have a complete backup so we use this helper function
		restoreErr := restoreBackup(
			liveCompose,
			liveNginx,
			backupCompose,
			backupNginx,
			backupRoot,
		)

		if restoreErr != nil {
			return "", errors.Join(
				fmt.Errorf("publish candidate compose: %w", err),
				fmt.Errorf("restore previous configuration: %w", restoreErr),
			)
		}

		return "", fmt.Errorf("publish candidate compose: %w", err)
	}

	if err := os.Rename(candidateNginx, liveNginx); err != nil {
		restoreErr := restoreBackup(
			liveCompose,
			liveNginx,
			backupCompose,
			backupNginx,
			backupRoot,
		)

		if restoreErr != nil {
			return "", errors.Join(
				fmt.Errorf("publish candidate nginx: %w", err),
				fmt.Errorf("restore previous configuration: %w", restoreErr),
			)
		}

		return "", fmt.Errorf("publish candidate nginx: %w", err)
	}

	// Keep this backup until Docker and Nginx runtime verification succeeds.
	return backupRoot, nil
}

func applyRuntime(liveRoot string) error {
	composeCmd := exec.Command(
		"docker",
		"compose",
		"up",
		"-d",
		"--remove-orphans",
	)
	composeCmd.Dir = liveRoot

	if output, err := composeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"apply compose runtime: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}
	//composeCmd won't update the running nginx config so we need to force restart on nginx
	nginxCmd := exec.Command(
		"docker",
		"compose",
		"up",
		"-d",
		"--force-recreate",
		"--no-deps",
		"nginx",
	)
	nginxCmd.Dir = liveRoot

	if output, err := nginxCmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"recreate nginx container: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return nil
}

// this is a wrapper function that calls restoreBackup using the variable s we have in main()
func rollbackPublishedConfigs(liveRoot, backupRoot string) error {
	return restoreBackup(
		filepath.Join(liveRoot, "compose.yaml"),
		filepath.Join(liveRoot, "nginx", "conf.d"),
		filepath.Join(backupRoot, "compose.yaml"),
		filepath.Join(backupRoot, "nginx", "conf.d"),
		backupRoot,
	)
}

func loadAppliedCommit(statePath string) (string, error) {
	//try to read state path, if its empty return empty string, if we get another error return it,
	//otherwise return the contents
	data, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read applied commit: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

func saveAppliedCommit(statePath, target string) error {
	stateDir := filepath.Dir(statePath)
	//make directory if it doesnt exist, with the path supplied by main()
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	//make temporary state file
	tempFile, err := os.CreateTemp(stateDir, "applied-*")
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}

	tempPath := tempFile.Name()
	//defer remove the temp file so if something errors it gets removed
	defer os.Remove(tempPath)
	//write applied commit sha to the temp file
	if _, err := tempFile.WriteString(target + "\n"); err != nil {
		tempFile.Close()
		return fmt.Errorf("write applied commit: %w", err)
	}
	//close the temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close applied state file: %w", err)
	}
	//rename to the statePath effectively overwriting it
	if err := os.Rename(tempPath, statePath); err != nil {
		return fmt.Errorf("publish applied state: %w", err)
	}

	return nil
}

func currentHeadCommit(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve current HEAD: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// return a map of path:SHA entries
func getTargetBlobs(repoPath string, target string) (map[string]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", target, "--format=%(objectname) %(path)", "--", "compose.yaml", "nginx/conf.d/")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list target blobs: %w", err)
	}
	shaMap := make(map[string]string)
	for line := range strings.Lines(string(output)) {
		cleanLine := strings.TrimSuffix(line, "\n")
		pair := strings.SplitN(cleanLine, " ", 2)
		if len(pair) == 1 {
			return nil, fmt.Errorf("unexpected ls-tree output: %q", cleanLine)
		}
		shaMap[pair[1]] = pair[0]
	}

	return shaMap, nil
}

func getAppliedBlobs(repoPath string) error {
	filepath.WalkDir(repoPath, getLocalBlob())
}
