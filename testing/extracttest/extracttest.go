// Package extracttest provides structures to help create tabular tests for extractors.
package extracttest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/osv-scalibr/extractor"
	"github.com/google/osv-scalibr/extractor/filesystem"
	scalibrfs "github.com/google/osv-scalibr/fs"
	"github.com/google/osv-scalibr/testing/fakefs"
	"github.com/google/osv-scalibr/testing/internal/inventorysorter"
)

// ScanInputMockConfig is used to quickly configure building a mock ScanInput
type ScanInputMockConfig struct {
	// Path of the file ScanInput will read, relative to the ScanRoot
	Path string
	// FakeScanRoot allows you to set a custom scanRoot, can be relative or absolute,
	// and will be translated to an absolute path
	FakeScanRoot string
	FakeFileInfo *fakefs.FakeFileInfo
}

// TestTableEntry is a entry to pass to ExtractionTester
type TestTableEntry struct {
	Name          string
	InputConfig   ScanInputMockConfig
	WantInventory []*extractor.Inventory
	WantErr       error
}

// ContainsErrStr is an error that matches other errors that contains
// `str` in their error string.
type ContainsErrStr struct {
	Str string
}

// Error returns the error string
func (e ContainsErrStr) Error() string { return fmt.Sprintf("error contains: '%s'", e.Str) }

// Is checks whether the input error contains the string in ContainsErrStr
func (e ContainsErrStr) Is(err error) bool {
	return strings.Contains(err.Error(), e.Str)
}

// ExtractionTester tests common properties of a extractor, and returns the raw values from running extract
func ExtractionTester(t *testing.T, extractor filesystem.Extractor, tt TestTableEntry) ([]*extractor.Inventory, error) {
	t.Helper()

	wrapper := generateScanInputMock(t, tt.InputConfig)
	got, err := extractor.Extract(context.Background(), &wrapper.ScanInput)
	wrapper.close()

	if !cmp.Equal(err, tt.WantErr, cmpopts.EquateErrors()) {
		t.Errorf(
			"%s.Extract(%s) error diff:\n%s",
			extractor.Name(),
			tt.InputConfig.Path,
			cmp.Diff(err, tt.WantErr, cmpopts.EquateErrors()),
		)
		return got, err
	}

	// Check if result match if no errors
	inventorysorter.Sort(got, extractor)
	inventorysorter.Sort(tt.WantInventory, extractor)
	if !cmp.Equal(got, tt.WantInventory) {
		t.Errorf("%s.Extract(%s) diff:\n%s", extractor.Name(), tt.InputConfig.Path, cmp.Diff(got, tt.WantInventory))
	}

	return got, err
}

type scanInputWrapper struct {
	fileHandle *os.File
	ScanInput  filesystem.ScanInput
}

func (siw scanInputWrapper) close() {
	siw.fileHandle.Close()
}

// generateScanInputMock will try to open the file locally, and fail if the file doesn't exist
func generateScanInputMock(t *testing.T, config ScanInputMockConfig) scanInputWrapper {
	t.Helper()

	var scanRoot string
	if filepath.IsAbs(config.FakeScanRoot) {
		scanRoot = config.FakeScanRoot
	} else {
		workingDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Can't get working directory because '%s'", workingDir)
		}
		scanRoot = filepath.Join(workingDir, config.FakeScanRoot)
	}

	f, err := os.Open(filepath.Join(scanRoot, config.Path))
	if err != nil {
		t.Fatalf("Can't open test fixture '%s' because '%s'", config.Path, err)
	}
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("Can't stat test fixture '%s' because '%s'", config.Path, err)
	}

	return scanInputWrapper{
		fileHandle: f,
		ScanInput: filesystem.ScanInput{
			FS:     os.DirFS(scanRoot).(scalibrfs.FS),
			Path:   config.Path,
			Root:   scanRoot,
			Reader: f,
			Info:   info,
		},
	}
}