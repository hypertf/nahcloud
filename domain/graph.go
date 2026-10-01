package domain

import "time"

type Network struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	Region    string    `json:"region"`
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
	Region    string    `json:"region"`
	Type      string    `json:"type"`
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
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Effect      string    `json:"effect"`
	Actions     []string  `json:"actions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type PolicyBinding struct {
	ID            string    `json:"id"`
	OrgID         string    `json:"org_id"`
	PolicyID      string    `json:"policy_id"`
	PrincipalType string    `json:"principal_type"`
	PrincipalID   string    `json:"principal_id"`
	TargetType    string    `json:"target_type"`
	TargetID      string    `json:"target_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
type LoadBalancer struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	Name            string    `json:"name"`
	SubnetID        string    `json:"subnet_id"`
	Region          string    `json:"region"`
	Protocol        string    `json:"protocol"`
	Port            int       `json:"port"`
	Algorithm       string    `json:"algorithm"`
	HealthCheckPath string    `json:"health_check_path"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
type LoadBalancerBackend struct {
	ID             string    `json:"id"`
	LoadBalancerID string    `json:"load_balancer_id"`
	InstanceID     string    `json:"instance_id"`
	Port           int       `json:"port"`
	Weight         int       `json:"weight"`
	Enabled        bool      `json:"enabled"`
	Healthy        bool      `json:"healthy"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateNetworkRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}
type UpdateNetworkRequest struct {
	Name *string `json:"name,omitempty"`
}
type CreateSubnetRequest struct {
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}
type UpdateSubnetRequest struct {
	Name *string `json:"name,omitempty"`
}
type CreateDiskRequest struct {
	Name   string `json:"name"`
	Region string `json:"region"`
	Type   string `json:"type"`
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
type CreatePolicyRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Effect      string   `json:"effect"`
	Actions     []string `json:"actions"`
}
type UpdatePolicyRequest struct {
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Effect      *string   `json:"effect,omitempty"`
	Actions     *[]string `json:"actions,omitempty"`
}
type CreatePolicyBindingRequest struct {
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	TargetType    string `json:"target_type"`
	TargetID      string `json:"target_id"`
}
type CreateLoadBalancerRequest struct {
	Name            string `json:"name"`
	SubnetID        string `json:"subnet_id"`
	Protocol        string `json:"protocol"`
	Port            int    `json:"port"`
	Algorithm       string `json:"algorithm"`
	HealthCheckPath string `json:"health_check_path"`
}
type UpdateLoadBalancerRequest struct {
	Name            *string `json:"name,omitempty"`
	Algorithm       *string `json:"algorithm,omitempty"`
	HealthCheckPath *string `json:"health_check_path,omitempty"`
}
type CreateLoadBalancerBackendRequest struct {
	InstanceID string `json:"instance_id"`
	Port       int    `json:"port"`
	Weight     int    `json:"weight"`
	Enabled    *bool  `json:"enabled,omitempty"`
}
type UpdateLoadBalancerBackendRequest struct {
	Port    *int  `json:"port,omitempty"`
	Weight  *int  `json:"weight,omitempty"`
	Enabled *bool `json:"enabled,omitempty"`
}
type PolicyEvaluationRequest struct {
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	Action        string `json:"action"`
	TargetType    string `json:"target_type"`
	TargetID      string `json:"target_id"`
}
type PolicyEvaluation struct {
	Result             string   `json:"result"`
	MatchingBindingIDs []string `json:"matching_binding_ids"`
}
