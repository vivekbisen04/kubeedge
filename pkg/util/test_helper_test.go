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
	// Test with a valid file
	tempFile, err := os.CreateTemp("", "test_config.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	_, err = tempFile.WriteString("test config content")
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	tempFile.Close()

	data, err := ReadConfigFile(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, []byte("test config content"), data)


	// Test with an invalid filename
	_, err = ReadConfigFile("")
	assert.Error(t, err)
	assert.EqualError(t, err, "filename cannot be empty")

	// Test with a non-existent file
	_, err = ReadConfigFile("nonexistent_file.txt")
	assert.Error(t, err) // os.ReadFile will return an error for a non-existent file.  Specific error check is OS-dependent.

}

func TestValidatePort(t *testing.T) {
	testCases := []struct {
		name        string
		port        int
		expectedErr bool
	}{
		{"ValidPort", 8080, false},
		{"PortZero", 0, true},
		{"PortTooHigh", 65536, true},
		{"PortNegative", -1, true},
		{"PortOne",1, false},
		{"PortMax", 65535, false},
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