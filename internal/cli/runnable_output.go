package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

const runnableStagedPackageName = "package.zip"

type runnableOutputStage struct {
	directory   string
	packagePath string
	finalPath   string
	published   bool
	cleanupDone bool
}

func resolveRunnableOutputPath(
	cwd string,
	requestedOutput string,
	manifestName string,
	manifestVersion string,
	targetOS string,
	targetArch string,
) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		return "", runnableOutputError("working directory is required", nil)
	}
	absoluteCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", runnableOutputError("resolve working directory", err)
	}
	cwdInfo, err := os.Stat(absoluteCWD)
	if err != nil {
		return "", runnableOutputError("stat working directory", err)
	}
	if !cwdInfo.IsDir() {
		return "", runnableOutputError("working directory is not a directory", nil)
	}

	var outputPath string
	if strings.TrimSpace(requestedOutput) == "" {
		for label, value := range map[string]string{
			"manifest name":       manifestName,
			"manifest version":    manifestVersion,
			"target OS":           targetOS,
			"target architecture": targetArch,
		} {
			if !validRunnableFilenameComponent(value) {
				return "", runnableOutputError(label+" is not a safe filename component", nil)
			}
		}

		outputPath = filepath.Join(
			absoluteCWD,
			"build",
			manifestName+"-"+manifestVersion+"-runnable-"+targetOS+"-"+targetArch+".zip",
		)
	} else {
		if filepath.Ext(requestedOutput) != ".zip" {
			return "", runnableOutputError("custom output path must have exact .zip extension", nil)
		}
		if filepath.IsAbs(requestedOutput) {
			outputPath = requestedOutput
		} else {
			outputPath = filepath.Join(absoluteCWD, requestedOutput)
		}
	}

	absoluteOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return "", runnableOutputError("resolve runnable output path", err)
	}
	if strings.TrimSpace(absoluteOutput) == "" || filepath.Ext(absoluteOutput) != ".zip" {
		return "", runnableOutputError("runnable output path is invalid", nil)
	}

	return absoluteOutput, nil
}

func validRunnableFilenameComponent(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	if !asciiLetterOrDigit(value[0]) || !asciiLetterOrDigit(value[len(value)-1]) {
		return false
	}

	for i := 1; i < len(value)-1; i++ {
		character := value[i]
		if asciiLetterOrDigit(character) || character == '.' || character == '_' ||
			character == '-' || character == '+' {
			continue
		}
		return false
	}

	return true
}

func asciiLetterOrDigit(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}

func prepareRunnableOutputStage(finalPath string) (*runnableOutputStage, error) {
	if strings.TrimSpace(finalPath) == "" || !filepath.IsAbs(finalPath) {
		return nil, runnableOutputError("final output path must be absolute", nil)
	}
	if filepath.Ext(finalPath) != ".zip" {
		return nil, runnableOutputError("final output path must have exact .zip extension", nil)
	}

	parent := filepath.Dir(finalPath)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, runnableOutputError("create runnable output parent", err)
	}
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return nil, runnableOutputError("stat runnable output parent", err)
	}
	if !parentInfo.IsDir() {
		return nil, runnableOutputError("runnable output parent is not a directory", nil)
	}

	if _, err := os.Lstat(finalPath); err == nil {
		return nil, runnableOutputError("final output path already exists", nil)
	} else if !os.IsNotExist(err) {
		return nil, runnableOutputError("inspect final output path", err)
	}

	directory, err := os.MkdirTemp(parent, ".forge-runnable-*")
	if err != nil {
		return nil, runnableOutputError("create runnable output staging directory", err)
	}
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(directory)
		if statErr != nil {
			_ = os.RemoveAll(directory)
			return nil, runnableOutputError("stat runnable output staging directory", statErr)
		}
		if info.Mode().Perm()&0o077 != 0 {
			_ = os.RemoveAll(directory)
			return nil, runnableOutputError("runnable output staging directory is not private", nil)
		}
	}

	return &runnableOutputStage{
		directory:   directory,
		packagePath: filepath.Join(directory, runnableStagedPackageName),
		finalPath:   finalPath,
	}, nil
}

func (s *runnableOutputStage) Publish() error {
	if s == nil || s.cleanupDone || strings.TrimSpace(s.directory) == "" ||
		strings.TrimSpace(s.packagePath) == "" || strings.TrimSpace(s.finalPath) == "" {
		return runnableOutputError("runnable output stage is invalid", nil)
	}
	if s.published {
		return nil
	}

	stagedInfo, err := os.Lstat(s.packagePath)
	if err != nil {
		return runnableOutputError("inspect staged runnable package", err)
	}
	if stagedInfo.Mode()&os.ModeSymlink != 0 || !stagedInfo.Mode().IsRegular() {
		return runnableOutputError("staged runnable package is not a regular file", nil)
	}

	if err := os.Link(s.packagePath, s.finalPath); err != nil {
		return runnableOutputError("publish runnable package without replacement", err)
	}

	finalInfo, err := os.Lstat(s.finalPath)
	if err != nil {
		return runnableOutputError("inspect published runnable package", err)
	}
	if finalInfo.Mode()&os.ModeSymlink != 0 || !finalInfo.Mode().IsRegular() {
		return runnableOutputError("published runnable package is not a regular file", nil)
	}
	if !os.SameFile(stagedInfo, finalInfo) {
		return runnableOutputError("published runnable package identity does not match staging", nil)
	}

	s.published = true
	return nil
}

func (s *runnableOutputStage) Cleanup() error {
	if s == nil || s.cleanupDone {
		return nil
	}
	if strings.TrimSpace(s.directory) == "" {
		return runnableOutputError("runnable output staging directory is invalid", nil)
	}

	if err := os.RemoveAll(s.directory); err != nil {
		return runnableOutputError("cleanup runnable output staging directory", err)
	}

	s.cleanupDone = true
	s.directory = ""
	s.packagePath = ""
	return nil
}

func runnableOutputError(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", compiler.ErrInvalidArtifactPackage, message)
	}

	return fmt.Errorf("%w: %s: %w", compiler.ErrInvalidArtifactPackage, message, cause)
}
