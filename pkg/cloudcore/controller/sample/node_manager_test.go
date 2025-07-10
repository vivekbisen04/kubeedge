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

package sample

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewNodeManager(t *testing.T) {
	nm := NewNodeManager()
	assert.NotNil(t, nm)
	assert.Equal(t, 0, len(nm.nodes))
	assert.WithinDuration(t, nm.lastSync, time.Now(), time.Second) //Allow for slight time difference
}

func TestRegisterNode(t *testing.T) {
	testCases := []struct {
		name        string
		nodeID      string
		nodeName    string
		expectedErr error
	}{
		{"Success", "node1", "Node 1", nil},
		{"Empty NodeID", "", "Node 2", fmt.Errorf("node ID cannot be empty")},
		{"Empty NodeName", "node3", "", fmt.Errorf("node name cannot be empty")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nm := NewNodeManager()
			err := nm.RegisterNode(context.Background(), tc.nodeID, tc.nodeName)
			assert.Equal(t, tc.expectedErr, err)
			if tc.expectedErr == nil {
				assert.Equal(t, 1, len(nm.nodes))
				assert.Equal(t, tc.nodeName, nm.nodes[tc.nodeID].Name)
			}
		})
	}
}

func TestGetNode(t *testing.T) {
	nm := NewNodeManager()
	nm.RegisterNode(context.Background(), "node1", "Node 1")

	t.Run("Success", func(t *testing.T) {
		node, err := nm.GetNode("node1")
		assert.NoError(t, err)
		assert.Equal(t, "node1", node.ID)
		assert.Equal(t, "Node 1", node.Name)
	})

	t.Run("NodeNotFound", func(t *testing.T) {
		_, err := nm.GetNode("node2")
		assert.Error(t, err)
		assert.EqualError(t, err, "node not found: node2")
	})
}

func TestListNodes(t *testing.T) {
	nm := NewNodeManager()
	nm.RegisterNode(context.Background(), "node1", "Node 1")
	nm.RegisterNode(context.Background(), "node2", "Node 2")

	nodes := nm.ListNodes()
	assert.Len(t, nodes, 2)
	assert.Equal(t, "node1", nodes[0].ID)
	assert.Equal(t, "node2", nodes[1].ID)
}


func TestUpdateNodeStatus(t *testing.T) {
	nm := NewNodeManager()
	nm.RegisterNode(context.Background(), "node1", "Node 1")

	t.Run("Success", func(t *testing.T) {
		err := nm.UpdateNodeStatus("node1", "offline")
		assert.NoError(t, err)
		assert.Equal(t, "offline", nm.nodes["node1"].Status)
		assert.WithinDuration(t, nm.nodes["node1"].LastSeen, time.Now(), time.Second)
	})

	t.Run("NodeNotFound", func(t *testing.T) {
		err := nm.UpdateNodeStatus("node2", "offline")
		assert.Error(t, err)
		assert.EqualError(t, err, "node not found: node2")
	})
}

func TestSyncNodes(t *testing.T) {
	nm := NewNodeManager()
	nm.RegisterNode(context.Background(), "node1", "Node 1")
	nm.nodes["node1"].LastSeen = time.Now().Add(-6 * time.Minute) //Simulate node offline

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	// Mock klog.Warningf to avoid output during testing
	patches.ApplyFunc(klog.Warningf, func(format string, args ...interface{}) {})

	err := nm.SyncNodes(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "offline", nm.nodes["node1"].Status)
	assert.WithinDuration(t, nm.lastSync, time.Now(), time.Second)
}

func TestGetNodeCount(t *testing.T) {
	nm := NewNodeManager()
	nm.RegisterNode(context.Background(), "node1", "Node 1")
	nm.RegisterNode(context.Background(), "node2", "Node 2")
	assert.Equal(t, 2, nm.GetNodeCount())
}

func TestRemoveNode(t *testing.T) {
	nm := NewNodeManager()
	nm.RegisterNode(context.Background(), "node1", "Node 1")

	t.Run("Success", func(t *testing.T) {
		err := nm.RemoveNode("node1")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(nm.nodes))
	})

	t.Run("NodeNotFound", func(t *testing.T) {
		err := nm.RemoveNode("node2")
		assert.Error(t, err)
		assert.EqualError(t, err, "node not found: node2")
	})
}