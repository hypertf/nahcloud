package domain

import "time"

type Network struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	CIDR      string    `json:"cidr"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Subnet struct {
	ID        string    `json:"id"`
	NetworkID string    `json:"network_id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	CIDR      string    `json:"cidr"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Disk struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	SizeGB    int       `json:"size_gb"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DiskAttachment struct {
	ID         string    `json:"id"`
	DiskID     string    `json:"disk_id"`
	ProjectID  string    `json:"project_id"`
	InstanceID string    `json:"instance_id"`
	Device     string    `json:"device"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Policy struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	ProjectID string    `json:"project_id,omitempty"`
	Name      string    `json:"name"`
	Document  string    `json:"document"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PolicyBinding struct {
	ID        string    `json:"id"`
	PolicyID  string    `json:"policy_id"`
	Principal string    `json:"principal"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LoadBalancer struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	Protocol  string    `json:"protocol"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LoadBalancerBackend struct {
	ID             string    `json:"id"`
	LoadBalancerID string    `json:"load_balancer_id"`
	ProjectID      string    `json:"project_id"`
	InstanceID     string    `json:"instance_id"`
	Port           int       `json:"port"`
	Weight         int       `json:"weight"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateNetworkRequest struct {
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}
type UpdateNetworkRequest struct {
	Name *string `json:"name,omitempty"`
	CIDR *string `json:"cidr,omitempty"`
}
type CreateSubnetRequest struct {
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}
type UpdateSubnetRequest struct {
	Name *string `json:"name,omitempty"`
	CIDR *string `json:"cidr,omitempty"`
}
type CreateDiskRequest struct {
	Name   string `json:"name"`
	SizeGB int    `json:"size_gb"`
}
type UpdateDiskRequest struct {
	Name   *string `json:"name,omitempty"`
	SizeGB *int    `json:"size_gb,omitempty"`
}
type CreateDiskAttachmentRequest struct {
	InstanceID string `json:"instance_id"`
	Device     string `json:"device"`
}
type UpdateDiskAttachmentRequest struct {
	Device *string `json:"device,omitempty"`
}
type CreatePolicyRequest struct {
	Name     string `json:"name"`
	Document string `json:"document"`
}
type UpdatePolicyRequest struct {
	Name     *string `json:"name,omitempty"`
	Document *string `json:"document,omitempty"`
}
type CreatePolicyBindingRequest struct {
	Principal string `json:"principal"`
	Role      string `json:"role"`
}
type UpdatePolicyBindingRequest struct {
	Principal *string `json:"principal,omitempty"`
	Role      *string `json:"role,omitempty"`
}
type CreateLoadBalancerRequest struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}
type UpdateLoadBalancerRequest struct {
	Name     *string `json:"name,omitempty"`
	Protocol *string `json:"protocol,omitempty"`
	Port     *int    `json:"port,omitempty"`
}
type CreateLoadBalancerBackendRequest struct {
	InstanceID string `json:"instance_id"`
	Port       int    `json:"port"`
	Weight     int    `json:"weight,omitempty"`
}
type UpdateLoadBalancerBackendRequest struct {
	Port   *int `json:"port,omitempty"`
	Weight *int `json:"weight,omitempty"`
}
