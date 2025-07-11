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

package flag

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestConfigValue_IsBoolFlag(t *testing.T) {
	var m ConfigValue
	assert.True(t, m.IsBoolFlag())
}

func TestConfigValue_Get(t *testing.T) {
	var m ConfigValue
	m = ConfigTrue
	assert.Equal(t, ConfigTrue, m.Get())
}

func TestConfigValue_Set(t *testing.T) {
	testCases := []struct {
		input    string
		expected ConfigValue
		err      bool
	}{
		{"true", ConfigTrue, false},
		{"false", ConfigFalse, false},
		{"1", ConfigTrue, false},
		{"0", ConfigFalse, false},
		{"invalid", ConfigFalse, true},
	}

	for _, tc := range testCases {
		var m ConfigValue
		err := m.Set(tc.input)
		if tc.err {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, m)
		}
	}
}

func TestConfigValue_String(t *testing.T) {
	assert.Equal(t, "true", ConfigTrue.String())
	assert.Equal(t, "false", ConfigFalse.String())
}

func TestConfigValue_Type(t *testing.T) {
	var m ConfigValue
	assert.Equal(t, "config", m.Type())
}

func TestConfigVar(t *testing.T) {
	var p ConfigValue
	ConfigVar(&p, "test", ConfigTrue, "test usage")
	assert.Equal(t, ConfigTrue, p)
	assert.NotNil(t, pflag.Lookup("test"))
	assert.Equal(t, "true", pflag.Lookup("test").NoOptDefVal)

}

func TestConfig(t *testing.T) {
	p := Config("test", ConfigFalse, "test usage")
	assert.NotNil(t, p)
	assert.Equal(t, ConfigFalse, *p)
	assert.NotNil(t, pflag.Lookup("test"))
	assert.Equal(t, "true", pflag.Lookup("test").NoOptDefVal)
}

func TestAddFlags(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ExitOnError)
	AddFlags(fs)
	assert.NotNil(t, fs.Lookup(minConfigFlagName))
	assert.NotNil(t, fs.Lookup(defaultConfigFlagName))
}


func TestPrintMinConfigAndExitIfRequested(t *testing.T) {
	// Mocking os.Exit
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(os.Exit, func(code int) {
	})

	// Test case: minConfigFlag is true
	*minConfigFlag = ConfigTrue
	config := map[string]string{"key": "value"}
	PrintMinConfigAndExitIfRequested(config)

	// Test case: minConfigFlag is false
	*minConfigFlag = ConfigFalse
	PrintMinConfigAndExitIfRequested(config)

}

func TestPrintDefaultConfigAndExitIfRequested(t *testing.T) {
	// Mocking os.Exit
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(os.Exit, func(code int) {})

	// Test case: defaultConfigFlag is true
	*defaultConfigFlag = ConfigTrue
	config := map[string]string{"key": "value"}
	PrintDefaultConfigAndExitIfRequested(config)

	// Test case: defaultConfigFlag is false
	*defaultConfigFlag = ConfigFalse
	PrintDefaultConfigAndExitIfRequested(config)
}

func TestPrintFlags(t *testing.T) {
	// Mocking klog.V
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	var logOutput string
	patches.ApplyFunc(klog.V, func(level int) *klog.Klog {
		return &klog.Klog{
			InfoDepth: func(format string, args ...interface{}) {
				logOutput = fmt.Sprintf(format, args...)
			},
		}
	})

	fs := pflag.NewFlagSet("test", pflag.ExitOnError)
	fs.String("test", "value", "test flag")
	PrintFlags(fs)
	assert.Contains(t, logOutput, "FLAG: --test=\"value\"")
}