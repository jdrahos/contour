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
	"testing"

	"github.com/google/go-cmp/cmp"
	logrus "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_cache "k8s.io/client-go/tools/cache"
)

func TestNodeWeightCache(t *testing.T) {
	tests := map[string]struct {
		initialState         []*v1.Node
		nodeName             string
		nodeWeightAnnotation string
		defaultNodeWeight    uint32
		nextAddCalled        bool
		nextUpdateCalled     bool
		nextDeleteCalled     bool
		old                  interface{}
		new                  interface{}
		want                 uint32
	}{
		"weight from annotation": {
			nextAddCalled:        true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			new: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "5",
					},
				},
			},
			want: 5,
		},
		"default weight if node name is missing": {
			nextAddCalled:        false,
			nodeName:             "",
			nodeWeightAnnotation: "weight-annotation",
			defaultNodeWeight:    1,
			want:                 1,
		},
		"update weight from annotation": {
			initialState: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Annotations: map[string]string{
							"weight-annotation": "10",
						},
					},
				},
			},
			nextUpdateCalled:     true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			new: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "5",
					},
				},
			},
			old: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "10",
					},
				},
			},
			want: 5,
		},
		"delete weight from annotation": {
			initialState: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Annotations: map[string]string{
							"weight-annotation": "5",
						},
					},
				},
			},
			nextDeleteCalled:     true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			old: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "5",
					},
				},
			},
			defaultNodeWeight: 1,
			want:              1,
		},
		"abnormal weight from annotation": {
			nextAddCalled:        true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			new: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "10000",
					},
				},
			},
			defaultNodeWeight: 1,
			want:              1,
		},
		"unparsable weight from annotation": {
			nextAddCalled:        true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			new: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "this will not parse as an int",
					},
				},
			},
			defaultNodeWeight: 1,
			want:              1,
		},
		"annotation not found": {
			nextAddCalled:        true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			new: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"not a weight-annotation": "5",
					},
				},
			},
			defaultNodeWeight: 1,
			want:              1,
		},
		"wrong type added": {
			nextAddCalled:        true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			new: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "10000",
					},
				},
			},
			defaultNodeWeight: 1,
			want:              1,
		},
		"wrong old type updated": {
			initialState: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Annotations: map[string]string{
							"weight-annotation": "10",
						},
					},
				},
			},
			nextUpdateCalled:     true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			new: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "5",
					},
				},
			},
			old: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "15",
					},
				},
			},
			defaultNodeWeight: 1,
			want:              5,
		},
		"wrong new type updated": {
			initialState: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Annotations: map[string]string{
							"weight-annotation": "5",
						},
					},
				},
			},
			nextUpdateCalled:     true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			new: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "15",
					},
				},
			},
			old: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "10",
					},
				},
			},
			want: 5,
		},
		"wrong type deleted": {
			initialState: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Annotations: map[string]string{
							"weight-annotation": "5",
						},
					},
				},
			},
			nextDeleteCalled:     true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			old: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Annotations: map[string]string{
						"weight-annotation": "5",
					},
				},
			},
			defaultNodeWeight: 1,
			want:              5,
		},

		"delete final state unknown": {
			initialState: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Annotations: map[string]string{
							"weight-annotation": "5",
						},
					},
				},
			},
			nextDeleteCalled:     true,
			nodeName:             "node1",
			nodeWeightAnnotation: "weight-annotation",
			old: _cache.DeletedFinalStateUnknown{
				Key: "node1",
				Obj: &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Annotations: map[string]string{
							"weight-annotation": "5",
						},
					},
				},
			},
			defaultNodeWeight: 1,
			want:              1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cache := NewNodeWeightCache(logrus.StandardLogger().WithField("context", "nodeHandler"), tc.nodeWeightAnnotation, tc.defaultNodeWeight)

			if tc.initialState != nil {
				for _, node := range tc.initialState {
					cache.OnAdd(node)
				}
			}

			resourceHandler := NewTestEventHandler()
			cache.Next = resourceHandler

			if tc.new != nil && tc.old != nil {
				cache.OnUpdate(tc.old, tc.new)
			} else if tc.new != nil {
				cache.OnAdd(tc.new)
			} else if tc.old != nil {
				cache.OnDelete(tc.old)
			}

			got := cache.GetWeightOfNode(tc.nodeName)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("Weight expected:\n%v\ngot:\n%v", tc.want, got)
			}
			if diff := cmp.Diff(tc.nextAddCalled, resourceHandler.addCalled); diff != "" {
				t.Fatalf("Handler add call expected:\n%v\ngot:\n%v", tc.nextAddCalled, resourceHandler.addCalled)
			}
			if diff := cmp.Diff(tc.nextUpdateCalled, resourceHandler.updateCalled); diff != "" {
				t.Fatalf("Handler update call expected:\n%v\ngot:\n%v", tc.nextUpdateCalled, resourceHandler.updateCalled)
			}
			if diff := cmp.Diff(tc.nextDeleteCalled, resourceHandler.deleteCalled); diff != "" {
				t.Fatalf("Handler delete call expected:\n%v\ngot:\n%v", tc.nextDeleteCalled, resourceHandler.deleteCalled)
			}
		})
	}
}

type TestEventHandler struct {
	addCalled    bool
	deleteCalled bool
	updateCalled bool
}

func NewTestEventHandler() *TestEventHandler {
	return &TestEventHandler{
		addCalled:    false,
		deleteCalled: false,
		updateCalled: false,
	}
}

func (h *TestEventHandler) OnAdd(_ interface{}) {
	h.addCalled = true
}

func (h *TestEventHandler) OnUpdate(_, _ interface{}) {
	h.updateCalled = true
}

func (h *TestEventHandler) OnDelete(_ interface{}) {
	h.deleteCalled = true
}
