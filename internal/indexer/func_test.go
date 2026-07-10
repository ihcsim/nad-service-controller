package indexer

import (
	"reflect"
	"testing"

	networkv1 "github.com/ihcsim/nad-service-controller/pkg/apis/k8s.cni.cncf.io/network/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceByNetworkFunc(t *testing.T) {
	testCases := []struct {
		desc     string
		obj      *corev1.Service
		expected []string
	}{
		{
			desc: "service with network annotation",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Annotations: map[string]string{
						ServiceNetworkAnnotation: "macvlan",
					},
				},
			},
			expected: []string{"default/macvlan"},
		},
		{
			desc: "service without network annotation",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   "default",
					Annotations: map[string]string{},
				},
			},
			expected: []string{},
		},
		{
			desc: "service with network annotation, custom namespace",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my_ns",
					Annotations: map[string]string{
						ServiceNetworkAnnotation: "macvlan",
					},
				},
			},
			expected: []string{"my_ns/macvlan"},
		},
		{
			desc: "service with network annotation, empty namespace",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ServiceNetworkAnnotation: "macvlan",
					},
				},
			},
			expected: []string{"macvlan"},
		},
		{
			desc:     "service with undefined annotation",
			obj:      &corev1.Service{},
			expected: []string{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			if actual := ServiceByNetworkFunc(testCase.obj); !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("result mismatch: expected %+v, got %+v", testCase.expected, actual)
			}
		})
	}
}

func TestPodByNetworkFunc(t *testing.T) {
	testCases := []struct {
		desc     string
		obj      *corev1.Pod
		expected []string
	}{
		{
			desc: "pod with one network",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						networkv1.NetworkStatusAnnot: `[{"name": "macvlan"}]`,
					},
				},
			},
			expected: []string{"macvlan"},
		},
		{
			desc: "pod with two networks",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						networkv1.NetworkStatusAnnot: `[{"name": "macvlan"}, {"name":"sriov-net"}]`,
					},
				},
			},
			expected: []string{"macvlan", "sriov-net"},
		},
		{
			desc: "pod with one detailed network",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						networkv1.NetworkStatusAnnot: `[
						{
            	"name": "default/macvlan",
            	"interface": "net3",
            	"ips": ["192.168.2.202"],
              "mac": "d2:97:24:44:b4:fd",
              "dns": {},
              "gateway": ["\u003cnil\u003e"]
            }
					]`,
					},
				},
			},
			expected: []string{"default/macvlan"},
		},
		{
			desc: "pod with two detailed networks",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						networkv1.NetworkStatusAnnot: `[
						{
            	"name": "default/macvlan",
            	"interface": "net3",
            	"ips": ["192.168.2.202"],
              "mac": "d2:97:24:44:b4:fd",
              "dns": {},
              "gateway": ["\u003cnil\u003e"]
            },
						{
            	"name": "demo/bridge-local",
            	"interface": "net1",
            	"ips": ["10.10.0.7"],
            	"mac": "4e:3a:ff:0f:46:03",
            	"dns": {},
            	"gateway": ["\u003cnil\u003e"]
        		}
					]`,
					},
				},
			},
			expected: []string{"default/macvlan", "demo/bridge-local"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			if actual := PodByNetworkFunc(testCase.obj); !reflect.DeepEqual(testCase.expected, actual) {
				t.Errorf("result mismatch: expected %+v, got %+v", testCase.expected, actual)
			}
		})
	}
}
