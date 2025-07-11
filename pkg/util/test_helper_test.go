/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"os"
)

func TestFormatMessage(t *testing.T) {
	testCases := []struct {
		name           string
		prefix         string
		message        string
		expectedOutput string
	}{
		{"EmptyPrefix", "", "test message", "test message"},
		{"NonEmptyPrefix", "my-prefix", "test message", "[my-prefix] test message"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := FormatMessage(tc.prefix, tc.message)
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}


func TestReadConfigFile(t *testing.T) {
	// Test with a valid filename
	tempFile, err := os.CreateTemp("", "test_config.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.WriteString("test config content")
	tempFile.Close()

	testCases := []struct {
		name        string
		filename    string
		expectedErr bool
	}{
		{"ValidFilename", tempFile.Name(), false},
		{"InvalidFilename", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := ReadConfigFile(tc.filename)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, data)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	testCases := []struct {
		name        string
		port        int
		expectedErr bool
	}{
		{"ValidPort1", 80, false},
		{"ValidPort2", 65535, false},
		{"InvalidPort1", 0, true},
		{"InvalidPort2", -1, true},
		{"InvalidPort3", 65536, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePort(tc.port)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}