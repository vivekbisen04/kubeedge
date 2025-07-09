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
		{
			name:           "Empty Prefix",
			prefix:         "",
			message:        "test message",
			expectedOutput: "test message",
		},
		{
			name:           "Non-Empty Prefix",
			prefix:         "myprefix",
			message:        "test message",
			expectedOutput: "[myprefix] test message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := FormatMessage(tc.prefix, tc.message)
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}


func TestReadConfigFile(t *testing.T) {
	// Positive test case: File exists
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


	// Negative test case: File does not exist
	_, err = ReadConfigFile("nonexistent_file.txt")
	assert.Error(t, err)

	//Negative test case: Empty filename
	_, err = ReadConfigFile("")
	assert.Error(t, err)

}

func TestValidatePort(t *testing.T) {
	testCases := []struct {
		name        string
		port        int
		expectedErr bool
	}{
		{
			name:        "Valid Port",
			port:        8080,
			expectedErr: false,
		},
		{
			name:        "Port Too Low",
			port:        0,
			expectedErr: true,
		},
		{
			name:        "Port Too High",
			port:        65536,
			expectedErr: true,
		},
		{
			name:        "Valid Port at lower bound",
			port:        1,
			expectedErr: false,
		},
		{
			name:        "Valid Port at upper bound",
			port:        65535,
			expectedErr: false,
		},

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