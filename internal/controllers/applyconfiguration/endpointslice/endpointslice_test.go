package endpointslice

import (
	"fmt"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	metatypes "k8s.io/apimachinery/pkg/types"

	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	discv1ac "k8s.io/client-go/applyconfigurations/discovery/v1"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"

	networkv1 "github.com/ihcsim/nad-service-controller/pkg/apis/k8s.cni.cncf.io/network/v1"

	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplyConfig(t *testing.T) {
	// this test case generates the ApplyConfig for an endpointslice of a service
	// backed by three pods, one of which is terminating. the ApplyConfig should
	// contain three endpoints, with IP addresses derived from the pods' network
	// annotations, and the conditions of each endpoint should reflect whether the
	// pod is ready, serving, or terminating. specifically, the endpoint
	// corresponding to the terminating pod should have its "terminating" condition
	// set to true, while the other two endpoints should have "ready" and "serving"
	// conditions set to true.

	now := metav1.Now()
	network := "default/macvlan"
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node",
			Labels: map[string]string{
				corev1.LabelTopologyZone: "us-east-1a",
			},
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
			UID:  "12345",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: corev1.ProtocolTCP,
					Port:     80,
				},
				{
					Name:     "https",
					Protocol: corev1.ProtocolTCP,
					Port:     443,
				},
			},
		},
	}
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-01",
				Namespace: "default",
				UID:       "1234567",
				Annotations: map[string]string{
					networkv1.NetworkStatusAnnot: `[
						{
            	"name": "default/macvlan",
            	"interface": "net3",
            	"ips": ["192.168.2.201"],
              "mac": "d2:97:24:44:b4:fd",
              "dns": {},
              "gateway": ["\u003cnil\u003e"]
        		}]`,
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node",
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-02",
				Namespace: "default",
				UID:       "89101112",
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
			Spec: corev1.PodSpec{
				NodeName: "node",
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "nginx-03",
				Namespace:         "default",
				DeletionTimestamp: &now,
				UID:               "13141516",
				Finalizers:        []string{"finalizer-to-for-fake-client"},
				Annotations: map[string]string{
					networkv1.NetworkStatusAnnot: `[
						{
            	"name": "default/macvlan",
            	"interface": "net3",
            	"ips": ["192.168.2.203"],
              "mac": "d2:97:24:44:b4:fd",
              "dns": {},
              "gateway": ["\u003cnil\u003e"]
        		}]`,
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node",
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
	}
	expected := &discv1ac.EndpointSliceApplyConfiguration{
		TypeMetaApplyConfiguration: metav1ac.TypeMetaApplyConfiguration{
			APIVersion: new("discovery.k8s.io/v1"),
			Kind:       new("EndpointSlice"),
		},
		ObjectMetaApplyConfiguration: &metav1ac.ObjectMetaApplyConfiguration{
			Name:      new("nginx-slice"),
			Namespace: new(""),
			Labels: map[string]string{
				discv1.LabelServiceName: svc.GetName(),
			},
			OwnerReferences: []metav1ac.OwnerReferenceApplyConfiguration{
				{
					Kind:       new("Service"),
					APIVersion: new(svc.APIVersion),
					Name:       new(svc.GetName()),
					UID:        new(svc.GetUID()),
				},
			},
		},
		AddressType: new(discv1.AddressTypeIPv4),
		Endpoints: []discv1ac.EndpointApplyConfiguration{
			{
				Addresses: []string{"192.168.2.201"},
				Conditions: &discv1ac.EndpointConditionsApplyConfiguration{
					Ready:       new(true),
					Serving:     new(true),
					Terminating: new(false),
				},
				NodeName: new(node.GetName()),
				TargetRef: &corev1ac.ObjectReferenceApplyConfiguration{
					Kind:      new("Pod"),
					Namespace: new("default"),
					Name:      new("nginx-01"),
					UID:       new(metatypes.UID("1234567")),
				},
				Zone: new("us-east-1a"),
			},
			{
				Addresses: []string{"192.168.2.202"},
				Conditions: &discv1ac.EndpointConditionsApplyConfiguration{
					Ready:       new(true),
					Serving:     new(true),
					Terminating: new(false),
				},
				NodeName: new(node.GetName()),
				TargetRef: &corev1ac.ObjectReferenceApplyConfiguration{
					Kind:      new("Pod"),
					Namespace: new("default"),
					Name:      new("nginx-02"),
					UID:       new(metatypes.UID("89101112")),
				},
				Zone: new("us-east-1a"),
			},
			{
				Addresses: []string{"192.168.2.203"},
				Conditions: &discv1ac.EndpointConditionsApplyConfiguration{
					Ready:       new(false),
					Serving:     new(false),
					Terminating: new(true),
				},
				NodeName: new(node.GetName()),
				TargetRef: &corev1ac.ObjectReferenceApplyConfiguration{
					Kind:      new("Pod"),
					Namespace: new("default"),
					Name:      new("nginx-03"),
					UID:       new(metatypes.UID("13141516")),
				},
				Zone: new("us-east-1a"),
			},
		},
		Ports: []discv1ac.EndpointPortApplyConfiguration{
			{
				Name:     new("http"),
				Protocol: new(corev1.ProtocolTCP),
				Port:     new(int32(80)),
			},
			{
				Name:     new("https"),
				Protocol: new(corev1.ProtocolTCP),
				Port:     new(int32(443)),
			},
		},
	}

	objs := []runtime.Object{svc, node}
	for _, pod := range pods {
		objs = append(objs, &pod)
	}
	client := fakeclient.NewFakeClient(objs...)
	actual, err := ApplyConfig(svc, pods, network, client)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("unexpected result: expected %+v,\ngot %+v", expected, actual)
	}
}

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
