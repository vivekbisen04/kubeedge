/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0
*/

package sample

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"
)

// NodeManager manages edge nodes in the cloud
type NodeManager struct {
	nodes    map[string]*EdgeNode
	lastSync time.Time
}

// EdgeNode represents an edge node
type EdgeNode struct {
	ID       string
	Name     string
	Status   string
	LastSeen time.Time
}

// NewNodeManager creates a new NodeManager instance
func NewNodeManager() *NodeManager {
	return &NodeManager{
		nodes:    make(map[string]*EdgeNode),
		lastSync: time.Now(),
	}
}

// RegisterNode registers a new edge node
func (nm *NodeManager) RegisterNode(ctx context.Context, nodeID, nodeName string) error {
	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	
	if nodeName == "" {
		return fmt.Errorf("node name cannot be empty")
	}

	nm.nodes[nodeID] = &EdgeNode{
		ID:       nodeID,
		Name:     nodeName,
		Status:   "online",
		LastSeen: time.Now(),
	}

	klog.Infof("Registered edge node: %s (%s)", nodeName, nodeID)
	return nil
}

// GetNode retrieves a node by ID
func (nm *NodeManager) GetNode(nodeID string) (*EdgeNode, error) {
	node, exists := nm.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}
	return node, nil
}

// ListNodes returns all registered nodes
func (nm *NodeManager) ListNodes() []*EdgeNode {
	var nodes []*EdgeNode
	for _, node := range nm.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// UpdateNodeStatus updates the status of a node
func (nm *NodeManager) UpdateNodeStatus(nodeID, status string) error {
	node, exists := nm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}
	
	node.Status = status
	node.LastSeen = time.Now()
	
	klog.Infof("Updated node %s status to %s", nodeID, status)
	return nil
}

// SyncNodes synchronizes all nodes
func (nm *NodeManager) SyncNodes(ctx context.Context) error {
	start := time.Now()
	defer func() {
		nm.lastSync = time.Now()
		klog.Infof("Node sync completed in %v", time.Since(start))
	}()

	// Simulate sync logic
	for nodeID, node := range nm.nodes {
		if time.Since(node.LastSeen) > 5*time.Minute {
			klog.Warningf("Node %s appears to be offline", nodeID)
			node.Status = "offline"
		}
	}

	return nil
}

// GetNodeCount returns the total number of registered nodes
func (nm *NodeManager) GetNodeCount() int {
	return len(nm.nodes)
}

// RemoveNode removes a node from management
func (nm *NodeManager) RemoveNode(nodeID string) error {
	_, exists := nm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}
	
	delete(nm.nodes, nodeID)
	klog.Infof("Removed node: %s", nodeID)
	return nil
}