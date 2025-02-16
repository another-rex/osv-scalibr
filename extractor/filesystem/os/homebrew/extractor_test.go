// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package homebrew_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/osv-scalibr/extractor"
	"github.com/google/osv-scalibr/extractor/filesystem"
	"github.com/google/osv-scalibr/extractor/filesystem/os/homebrew"
	"github.com/google/osv-scalibr/purl"
)

func TestFileRequired(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		wantIsRequired bool
	}{
		{
			name:           "wrong.path.json",
			path:           "testdata/otherfile.json",
			wantIsRequired: false,
		},
		{
			name:           "cellar.valid.json",
			path:           "testdata/Cellar/rclone/1.67.0/INSTALL_RECEIPT.json",
			wantIsRequired: true,
		},
		{
			name:           "cellar.invalid.json",
			path:           "testdata/Cellar/rclone/1.67.0/other.json",
			wantIsRequired: false,
		},
		{
			name:           "cellar.invalid.json2",
			path:           "testdata/Cellar/rclone/1.67.0/INSTALL_RECEIPT.json2",
			wantIsRequired: false,
		},
		{
			name:           "caskroom.valid.json",
			path:           "testdata/Caskroom/testapp/1.1.1/testapp.wrapper.sh",
			wantIsRequired: true,
		},
		{
			name:           "regex.invalid.json",
			path:           "testdata/Caskroom/testapp/1.1.1/testapp.app/Contents/PkgInfo",
			wantIsRequired: false,
		},
		{
			name:           "caskroom.other.variation",
			path:           "testdata/Caskroom/android-platform-tools/35.0.2/platform-tools/source.properties",
			wantIsRequired: true,
		},
		{
			name:           "caskroom.null.folder",
			path:           "testdata/Caskroom/somefolder/1.1/",
			wantIsRequired: false,
		},
		{
			name:           "caskroom.null.variation",
			path:           "testdata/Caskroom/somefolder/2.2",
			wantIsRequired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e filesystem.Extractor = homebrew.Extractor{}
			if got := e.FileRequired(tt.path, nil); got != tt.wantIsRequired {
				t.Fatalf("FileRequired(%s): got %v, want %v", tt.path, got, tt.wantIsRequired)
			}
		})
	}
}

func invLess(i1, i2 *extractor.Inventory) bool {
	return i1.Name < i2.Name
}

func TestExtract(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantErr       error
		wantInventory []*extractor.Inventory
	}{
		{
			name: "cellar.valid.json",
			path: "testdata/Cellar/rclone/1.67.0/INSTALL_RECEIPT.json",
			wantInventory: []*extractor.Inventory{
				{
					Name:      "rclone",
					Version:   "1.67.0",
					Locations: []string{"testdata/Cellar/rclone/1.67.0/INSTALL_RECEIPT.json"},
				},
			},
		},
		{
			name: "caskroom.valid.json",
			path: "testdata/Caskroom/testapp/1.1.1/testapp.wrapper.sh",
			wantInventory: []*extractor.Inventory{
				{
					Name:      "testapp",
					Version:   "1.1.1",
					Locations: []string{"testdata/Caskroom/testapp/1.1.1/testapp.wrapper.sh"},
				},
			},
		},
		{
			name:          "caskroom.null.variation",
			path:          "testdata/Caskroom/somefolder/2.2",
			wantInventory: []*extractor.Inventory{},
		},
		{
			name: "caskroom.other.variation",
			path: "testdata/Caskroom/android-platform-tools/35.0.2/platform-tools/source.properties",
			wantInventory: []*extractor.Inventory{
				{
					Name:      "android-platform-tools",
					Version:   "35.0.2",
					Locations: []string{"testdata/Caskroom/android-platform-tools/35.0.2/platform-tools/source.properties"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e filesystem.Extractor = homebrew.Extractor{}
			input := &filesystem.ScanInput{Path: tt.path, Reader: nil}
			got, err := e.Extract(context.Background(), input)
			if diff := cmp.Diff(tt.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Extract(%s) unexpected error (-want +got):\n%s", tt.path, diff)
			}

			want := tt.wantInventory

			if diff := cmp.Diff(want, got, cmpopts.SortSlices(invLess)); diff != "" {
				t.Errorf("Extract(%s) (-want +got):\n%s", tt.path, diff)
			}
		})
	}
}

func TestToPURL(t *testing.T) {
	tests := []struct {
		name      string
		inventory []*extractor.Inventory
		want      *purl.PackageURL
	}{
		{
			name: "cask_firefox",
			inventory: []*extractor.Inventory{
				&extractor.Inventory{
					Name:      "firefox",
					Version:   "129.0",
					Locations: []string{"System/Volumes/Data/usr/local/Caskroom/firefox/129.0/firefox.wrapper.sh"},
				},
			},
			want: &purl.PackageURL{
				Type:    purl.TypeBrew,
				Name:    "firefox",
				Version: "129.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e filesystem.Extractor = homebrew.Extractor{}
			for _, i := range tt.inventory {
				got, err := e.ToPURL(i)
				if err != nil {
					t.Fatalf("ToPURL(%v): %v", i, err)
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("ToPURL(%v) (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want *homebrew.BrewPath
	}{
		{
			name: "cellar_path",
			path: "testdata/Cellar/rclone/1.67.0/INSTALL_RECEIPT.json",
			want: &homebrew.BrewPath{
				AppName:    "rclone",
				AppVersion: "1.67.0",
				AppFile:    "install_receipt.json",
				AppExt:     "install_receipt.json",
			},
		},
		{
			name: "caskroom_path",
			path: "testdata/Caskroom/testapp/1.1.1/testapp.wrapper.sh",
			want: &homebrew.BrewPath{
				AppName:    "testapp",
				AppVersion: "1.1.1",
				AppFile:    "testapp.wrapper.sh",
				AppExt:     "testapp.wrapper.sh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := homebrew.SplitPath(tt.path)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SpiltPath(%v) (-want +got):\n%s", tt.path, diff)
			}
		})
	}
}
