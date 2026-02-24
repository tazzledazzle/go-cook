package v1

type TypeMeta struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

type ObjectMeta struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	UID               string            `json:"uid,omitempty"`
	ResourceVersion   string            `json:"resourceVersion,omitempty"`
	CreationTimestamp string            `json:"creationTimestamp,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
}

// POD

type ResourceRequirements struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

type Container struct {
	name      string                `json:"name"`
	Image     string                `json:"image"`
	Resources *ResourceRequirements `json:"resources,omitempty"`
}

type PodSpec struct {
	Containers []Container `json:"containers"`
	NodeName   string      `json:"nodeName,omitempty"`
}

type PodCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type PodStatus struct {
	Phase     string         `json:"phase,omitempty"`
	Condition []PodCondition `json:"condition,omitempty"`
	PodIP     string         `json:"podIP,omitempty"`
}

type Pod struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PodSpec   `json:"spec"`
	Status     PodStatus `json:"status,omitempty"`
}

// NODE

type Node struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       NodeSpec   `json:"spec"`
	Status     NodeStatus `json:"status,omitempty"`
}

type NodeSpec struct{} // todo

type NodeStatus struct {
	Capacity    map[string]string `json:"capacity,omitempty"`
	Allocatable map[string]string `json:"allocatable,omitempty"`
	Conditions  []NodeCondition   `json:"conditions,omitempty"`
	Addresses   []NodeAddress     `json:"addresses,omitempty"`
}

type NodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type NodeCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

// NAMESPACE

type NamespaceSpec struct{}

type NamespaceStatus struct {
	Phase string `json:"phase,omitempty"`
}
type Namespace struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata"`
	Spec       NamespaceSpec   `json:"spec"`
	Status     NamespaceStatus `json:"status,omitempty"`
}
