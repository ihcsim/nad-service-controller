package v1

import "net"

const (
	// Pod annotation for network-attachment-definition
	NetworkAttachmentAnnot = "k8s.v1.cni.cncf.io/networks"
	// Pod annotation for network status
	NetworkStatusAnnot = "k8s.v1.cni.cncf.io/network-status"
)

// PortMapEntry for CNI PortMapEntry
// +k8s:deepcopy-gen=false
type PortMapEntry struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
	HostIP        string `json:"hostIP,omitempty"`
}

// BandwidthEntry for CNI BandwidthEntry
// +k8s:deepcopy-gen=false
type BandwidthEntry struct {
	IngressRate  int `json:"ingressRate"`
	IngressBurst int `json:"ingressBurst"`

	EgressRate  int `json:"egressRate"`
	EgressBurst int `json:"egressBurst"`
}

// NetworkSelectionElement represents one element of the JSON format
// Network Attachment Selection Annotation as described in section 4.1.2
// of the CRD specification.
// +k8s:deepcopy-gen=false
type NetworkSelectionElement struct {
	// Name contains the name of the Network object this element selects
	Name string `json:"name"`
	// Namespace contains the optional namespace that the network referenced
	// by Name exists in
	Namespace string `json:"namespace,omitempty"`
	// IPRequest contains an optional requested IP addresses for this network
	// attachment
	IPRequest []string `json:"ips,omitempty"`
	// MacRequest contains an optional requested MAC address for this
	// network attachment
	MacRequest string `json:"mac,omitempty"`
	// InfinibandGUIDRequest contains an optional requested Infiniband GUID
	// address for this network attachment
	InfinibandGUIDRequest string `json:"infiniband-guid,omitempty"`
	// InterfaceRequest contains an optional requested name for the
	// network interface this attachment will create in the container
	InterfaceRequest string `json:"interface,omitempty"`
	// PortMappingsRequest contains an optional requested port mapping
	// for the network
	PortMappingsRequest []*PortMapEntry `json:"portMappings,omitempty"`
	// BandwidthRequest contains an optional requested bandwidth for
	// the network
	BandwidthRequest *BandwidthEntry `json:"bandwidth,omitempty"`
	// CNIArgs contains additional CNI arguments for the network interface
	CNIArgs *map[string]interface{} `json:"cni-args,omitempty"`
	// GatewayRequest contains default route IP address for the pod
	GatewayRequest []net.IP `json:"default-route,omitempty"`
	// IPAMClaimReference container the IPAMClaim name where the IPs for this
	// attachment will be located.
	IPAMClaimReference string `json:"ipam-claim-reference,omitempty"`
}
