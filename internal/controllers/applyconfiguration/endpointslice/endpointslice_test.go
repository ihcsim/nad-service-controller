package endpointslice

import (
	"fmt"
	"reflect"
	"testing"

	networkv1 "github.com/ihcsim/nad-service-controller/pkg/apis/k8s.cni.cncf.io/network/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodTopology(t *testing.T) {
	testCases := []struct {
		desc     string
		node     *corev1.Node
		expected string
	}{
		{
			desc: "node with zone label",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						corev1.LabelTopologyZone: "us-east-1a",
					},
				},
			},
			expected: "us-east-1a",
		},
		{
			desc: "node without zone label",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
			},
			expected: "",
		},
		{
			desc: "node with undefined label map",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
			},
			expected: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			if actual := podTopology(testCase.node); actual != testCase.expected {
				t.Errorf("unexpected result: expected '%s', got '%s'", testCase.expected, actual)
			}
		})
	}
}

func TestReadiness(t *testing.T) {
	now := metav1.Now()
	testCases := []struct {
		desc                string
		pod                 *corev1.Pod
		svc                 *corev1.Service
		expectedReady       bool
		expectedServing     bool
		expectedTerminating bool
	}{
		{
			desc: "pod is serving and ready, and not terminating",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			svc:                 &corev1.Service{},
			expectedReady:       true,
			expectedServing:     true,
			expectedTerminating: false,
		},
		{
			desc: "pod is serving and terminating, but not ready",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &now,
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			svc:                 &corev1.Service{},
			expectedReady:       false,
			expectedServing:     true,
			expectedTerminating: true,
		},
		{
			desc: "pod is not ready and not terminating",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			svc:                 &corev1.Service{},
			expectedReady:       false,
			expectedServing:     false,
			expectedTerminating: false,
		},
		{
			desc: "pod is missing ready condition",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{},
			},
			svc:                 &corev1.Service{},
			expectedReady:       false,
			expectedServing:     false,
			expectedTerminating: false,
		},
		{
			desc:                "pod is missing status",
			pod:                 &corev1.Pod{},
			svc:                 &corev1.Service{},
			expectedReady:       false,
			expectedServing:     false,
			expectedTerminating: false,
		},
		{
			desc: "pod is terminating",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &now,
				},
			},
			svc:                 &corev1.Service{},
			expectedReady:       false,
			expectedServing:     false,
			expectedTerminating: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			if actualServing, actualReady, actualTerminating := readiness(testCase.pod, testCase.svc); actualServing != testCase.expectedServing || actualReady != testCase.expectedReady || actualTerminating != testCase.expectedTerminating {
				t.Errorf("unexpected result: expected (serving: %v, ready: %v, terminating: %v), got (serving: %v, ready: %v, terminating: %v)", testCase.expectedServing, testCase.expectedReady, testCase.expectedTerminating, actualServing, actualReady, actualTerminating)
			}
		})
	}
}

func TestPodNetworkIPs(t *testing.T) {
	testCases := []struct {
		desc     string
		pod      *corev1.Pod
		network  string
		expected []string
		err      error
	}{
		{
			desc: "pod with one network ip",
			pod: &corev1.Pod{
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
        		}]`,
					},
				},
			},
			network:  "default/macvlan",
			expected: []string{"192.168.2.202"},
		},
		{
			desc: "pod with two network ips",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						networkv1.NetworkStatusAnnot: `[
						{
            	"name": "default/macvlan",
            	"interface": "net3",
							"ips": ["192.168.2.202", "192.168.2.204"],
              "mac": "d2:97:24:44:b4:fd",
              "dns": {},
              "gateway": ["\u003cnil\u003e"]
        		}]`,
					},
				},
			},
			network:  "default/macvlan",
			expected: []string{"192.168.2.202", "192.168.2.204"},
		},
		{
			desc: "pod with two networks",
			pod: &corev1.Pod{
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
              "name": "default/bridge-local",
              "interface": "net1",
              "ips": ["10.10.0.7"],
              "mac": "4e:3a:ff:0f:46:03",
              "dns": {},
              "gateway": ["\u003cnil\u003e"]
            }]`,
					},
				},
			},
			network:  "default/bridge-local",
			expected: []string{"10.10.0.7"},
		},
		{
			desc: "pod with invalid JSON in network annotation",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						networkv1.NetworkStatusAnnot: `[
						{
            	"name": "default/macvlan",
            	"interface"
        		}]`,
					},
				},
			},
			network: "default/bridge-local",
			err:     fmt.Errorf("invalid JSON in pod %s annotation %s", "", networkv1.NetworkStatusAnnot),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			actual, err := podNetworkIPs(testCase.pod, testCase.network)
			if testCase.err != nil && fmt.Sprintf("%s", err) != fmt.Sprintf("%s", testCase.err) {
				t.Errorf("unexpected error: expected %v, got %v", testCase.err, err)
			}
			if !reflect.DeepEqual(actual, testCase.expected) {
				t.Errorf("unexpected result: expected %v, got %v", testCase.expected, actual)
			}
		})
	}
}
