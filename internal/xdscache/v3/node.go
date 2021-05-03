// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v3

import (
	"strconv"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type NodeWeightFunc func(string) uint32

type NodeWeightCache struct {
	logrus.FieldLogger
	mu sync.Mutex // protect cache mutations.

	Next                 cache.ResourceEventHandler
	NodeWeightAnnotation string
	DefaultNodeWeight    uint32
	nodeWeights          map[string]uint32
}

func NewNodeWeightCache(fieldLogger *logrus.Entry, nodeWeightAnnotation string, defaultNodeWeight uint32) *NodeWeightCache {
	return &NodeWeightCache{
		FieldLogger:          fieldLogger,
		NodeWeightAnnotation: nodeWeightAnnotation,
		DefaultNodeWeight:    defaultNodeWeight,
		nodeWeights:          map[string]uint32{},
	}
}

// GetWeightOfNode call to get the weight of node by supplying node's name
func (c *NodeWeightCache) GetWeightOfNode(nodeName string) uint32 {
	nodeWeight := c.nodeWeights[nodeName]
	if nodeWeight == 0 {
		return c.DefaultNodeWeight
	}
	return nodeWeight
}

func (c *NodeWeightCache) OnAdd(obj interface{}) {
	switch obj := obj.(type) {
	case *v1.Node:
		c.updateNodeWeight(obj)
	default:
		c.Errorf("OnAdd unexpected type %T: %#v", obj, obj)
	}
	if c.Next != nil {
		c.Next.OnAdd(obj)
	}
}

func (c *NodeWeightCache) OnUpdate(oldObj, newObj interface{}) {
	switch newObj := newObj.(type) {
	case *v1.Node:
		if !cmp.Equal(oldObj, newObj, cmpopts.IgnoreFields(v1.Node{}, "Status")) {
			c.updateNodeWeight(newObj)
		}
	default:
		c.Errorf("OnUpdate unexpected type %T: %#v", newObj, newObj)
	}
	if c.Next != nil {
		c.Next.OnUpdate(oldObj, newObj)
	}
}

func (c *NodeWeightCache) OnDelete(obj interface{}) {
	switch obj := obj.(type) {
	case *v1.Node:
		//just delete the node weight from cache, no endpoints should be running on the node so nothing else needs to be done
		c.deleteNodeWeight(obj)
	case cache.DeletedFinalStateUnknown:
		c.OnDelete(obj.Obj) // get the actual object if we get object in unknown final state
	default:
		c.Errorf("OnDelete unexpected type %T: %#v", obj, obj)
	}
	if c.Next != nil {
		c.Next.OnDelete(obj)
	}
}

func (c *NodeWeightCache) updateNodeWeight(node *v1.Node) {
	c.mu.Lock()
	defer c.mu.Unlock()

	weight := resolveNodeWeight(node.ObjectMeta, c.NodeWeightAnnotation, c.DefaultNodeWeight)
	previousWeight := c.nodeWeights[node.Name]
	changed := previousWeight != weight

	if changed {
		c.nodeWeights[node.Name] = weight
	}
}

func (c *NodeWeightCache) deleteNodeWeight(node *v1.Node) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.nodeWeights, node.Name)
}

func resolveNodeWeight(meta metav1.ObjectMeta, annotationName string, defaultValue uint32) uint32 {
	weight := defaultValue

	if annotationStringValue, ok := meta.Annotations[annotationName]; ok {
		if nweight, cerr := strconv.ParseUint(annotationStringValue, 10, 32); cerr == nil {
			weight = uint32(nweight)
		}
	}

	return normalizeWeight(weight, defaultValue)
}

func normalizeWeight(weight, defaultWeight uint32) uint32 {
	if weight > 128 {
		return defaultWeight
	}
	return weight
}
