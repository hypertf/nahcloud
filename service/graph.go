package service

import (
	"net"
	"regexp"
	"slices"
	"strings"

	"github.com/hypertf/nahcloud/domain"
)

func graphID() (string, error)       { return generateID() }
func validRegion(region string) bool { return slices.Contains(domain.ValidRegions, region) }
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return domain.InvalidInputError("port must be between 1 and 65535", nil)
	}
	return nil
}
func (s *Service) validateProject(id string) (*domain.Project, error) {
	v, e := s.projectRepo.GetByID(id)
	if domain.IsNotFound(e) {
		return nil, domain.ForeignKeyViolationError("project", "id", id)
	}
	return v, e
}

func (s *Service) CreateNetwork(project string, req domain.CreateNetworkRequest) (*domain.Network, error) {
	if _, e := s.validateProject(project); e != nil {
		return nil, e
	}
	if e := validateName(req.Name, "network"); e != nil {
		return nil, e
	}
	if !validRegion(req.Region) {
		return nil, domain.InvalidInputError("invalid network region", nil)
	}
	id, e := graphID()
	if e != nil {
		return nil, domain.InternalError("generate ID")
	}
	v := &domain.Network{ID: id, ProjectID: project, Name: req.Name, Region: req.Region}
	if e = s.graphRepo.CreateNetwork(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) GetNetwork(id string) (*domain.Network, error) { return s.graphRepo.GetNetwork(id) }
func (s *Service) ListNetworks(project string) ([]*domain.Network, error) {
	return s.graphRepo.ListNetworks(project)
}
func (s *Service) UpdateNetwork(id string, req domain.UpdateNetworkRequest) (*domain.Network, error) {
	v, e := s.GetNetwork(id)
	if e != nil {
		return nil, e
	}
	if req.Name != nil {
		if e = validateName(*req.Name, "network"); e != nil {
			return nil, e
		}
		v.Name = *req.Name
	}
	if e = s.graphRepo.UpdateNetwork(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) DeleteNetwork(id string) error { return s.graphRepo.DeleteNetwork(id) }

func canonicalSubnet(cidr string) (*net.IPNet, error) {
	ip, network, e := net.ParseCIDR(cidr)
	if e != nil || ip.To4() == nil || network.String() != cidr || !ip.IsPrivate() {
		return nil, domain.InvalidInputError("cidr must be canonical private IPv4", nil)
	}
	prefix, bits := network.Mask.Size()
	if bits != 32 || prefix < 16 || prefix > 28 {
		return nil, domain.InvalidInputError("cidr prefix must be between /16 and /28", nil)
	}
	return network, nil
}
func overlaps(a, b *net.IPNet) bool { return a.Contains(b.IP) || b.Contains(a.IP) }
func (s *Service) CreateSubnet(network *domain.Network, req domain.CreateSubnetRequest) (*domain.Subnet, error) {
	parent, e := s.graphRepo.GetNetwork(network.ID)
	if e != nil {
		return nil, domain.NotFoundError("network", network.ID)
	}
	if e = validateName(req.Name, "subnet"); e != nil {
		return nil, e
	}
	candidate, e := canonicalSubnet(req.CIDR)
	if e != nil {
		return nil, e
	}
	existing, e := s.graphRepo.ListProjectSubnets(parent.ProjectID)
	if e != nil {
		return nil, e
	}
	for _, other := range existing {
		otherNetwork, e := s.graphRepo.GetNetwork(other.NetworkID)
		if e != nil {
			return nil, e
		}
		if otherNetwork.Region != parent.Region {
			continue
		}
		otherCIDR, _ := canonicalSubnet(other.CIDR)
		if overlaps(candidate, otherCIDR) {
			return nil, domain.AlreadyExistsError("subnet", "cidr", req.CIDR)
		}
	}
	id, e := graphID()
	if e != nil {
		return nil, e
	}
	v := &domain.Subnet{ID: id, NetworkID: parent.ID, ProjectID: parent.ProjectID, Name: req.Name, CIDR: req.CIDR}
	if e = s.graphRepo.CreateSubnet(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) GetSubnet(id string) (*domain.Subnet, error) { return s.graphRepo.GetSubnet(id) }
func (s *Service) ListSubnets(network string) ([]*domain.Subnet, error) {
	return s.graphRepo.ListSubnets(network)
}
func (s *Service) UpdateSubnet(id string, req domain.UpdateSubnetRequest) (*domain.Subnet, error) {
	v, e := s.GetSubnet(id)
	if e != nil {
		return nil, e
	}
	if req.Name != nil {
		if e = validateName(*req.Name, "subnet"); e != nil {
			return nil, e
		}
		v.Name = *req.Name
	}
	if e = s.graphRepo.UpdateSubnet(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) DeleteSubnet(id string) error { return s.graphRepo.DeleteSubnet(id) }

func (s *Service) validateSubnetForInstance(subnetID, project, region string) error {
	subnet, e := s.graphRepo.GetSubnet(subnetID)
	if e != nil || subnet.ProjectID != project {
		return domain.NotFoundError("subnet", subnetID)
	}
	network, e := s.graphRepo.GetNetwork(subnet.NetworkID)
	if e != nil || network.Region != region {
		return domain.InvalidInputError("subnet and instance regions must match", nil)
	}
	return nil
}

func (s *Service) CreateDisk(project string, req domain.CreateDiskRequest) (*domain.Disk, error) {
	if _, e := s.validateProject(project); e != nil {
		return nil, e
	}
	if e := validateName(req.Name, "disk"); e != nil {
		return nil, e
	}
	if !validRegion(req.Region) {
		return nil, domain.InvalidInputError("invalid disk region", nil)
	}
	if req.Type != "standard" && req.Type != "ssd" {
		return nil, domain.InvalidInputError("disk type must be standard or ssd", nil)
	}
	if req.SizeGB < 1 || req.SizeGB > 16384 {
		return nil, domain.InvalidInputError("size_gb must be between 1 and 16384", nil)
	}
	id, e := graphID()
	if e != nil {
		return nil, e
	}
	v := &domain.Disk{ID: id, ProjectID: project, Name: req.Name, Region: req.Region, Type: req.Type, SizeGB: req.SizeGB}
	if e = s.graphRepo.CreateDisk(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) GetDisk(id string) (*domain.Disk, error) { return s.graphRepo.GetDisk(id) }
func (s *Service) ListDisks(project string) ([]*domain.Disk, error) {
	return s.graphRepo.ListDisks(project)
}
func (s *Service) UpdateDisk(id string, req domain.UpdateDiskRequest) (*domain.Disk, error) {
	v, e := s.GetDisk(id)
	if e != nil {
		return nil, e
	}
	if req.Name != nil {
		if e = validateName(*req.Name, "disk"); e != nil {
			return nil, e
		}
		v.Name = *req.Name
	}
	if req.SizeGB != nil {
		if *req.SizeGB < v.SizeGB {
			return nil, domain.InvalidInputError("disk size cannot be decreased", nil)
		}
		if *req.SizeGB > 16384 {
			return nil, domain.InvalidInputError("size_gb must not exceed 16384", nil)
		}
		v.SizeGB = *req.SizeGB
	}
	if e = s.graphRepo.UpdateDisk(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) DeleteDisk(id string) error { return s.graphRepo.DeleteDisk(id) }

var devicePattern = regexp.MustCompile(`^vd[b-z]$`)

func (s *Service) CreateAttachment(disk *domain.Disk, req domain.CreateDiskAttachmentRequest) (*domain.DiskAttachment, error) {
	parent, e := s.GetDisk(disk.ID)
	if e != nil {
		return nil, domain.NotFoundError("disk", disk.ID)
	}
	if !devicePattern.MatchString(req.Device) {
		return nil, domain.InvalidInputError("device must match vd[b-z]", nil)
	}
	instance, e := s.instanceRepo.GetByID(req.InstanceID)
	if e != nil || instance.ProjectID != parent.ProjectID || instance.Region != parent.Region {
		return nil, domain.NotFoundError("instance", req.InstanceID)
	}
	id, e := graphID()
	if e != nil {
		return nil, e
	}
	v := &domain.DiskAttachment{ID: id, DiskID: parent.ID, ProjectID: parent.ProjectID, InstanceID: req.InstanceID, Device: req.Device}
	if e = s.graphRepo.CreateAttachment(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) GetAttachment(id string) (*domain.DiskAttachment, error) {
	return s.graphRepo.GetAttachment(id)
}
func (s *Service) ListAttachments(disk string) ([]*domain.DiskAttachment, error) {
	return s.graphRepo.ListAttachments(disk)
}
func (s *Service) DeleteAttachment(id string) error { return s.graphRepo.DeleteAttachment(id) }

func validatePolicy(effect string, actions []string) error {
	if effect != "allow" && effect != "deny" {
		return domain.InvalidInputError("effect must be allow or deny", nil)
	}
	if len(actions) < 1 || len(actions) > 32 {
		return domain.InvalidInputError("actions must contain 1 to 32 values", nil)
	}
	seen := map[string]bool{}
	for _, action := range actions {
		if strings.TrimSpace(action) == "" || seen[action] {
			return domain.InvalidInputError("actions must be non-empty and unique", nil)
		}
		seen[action] = true
	}
	return nil
}
func (s *Service) CreatePolicy(org, _ string, req domain.CreatePolicyRequest) (*domain.Policy, error) {
	if _, e := s.orgRepo.GetByID(org); e != nil {
		return nil, domain.ForeignKeyViolationError("organization", "id", org)
	}
	if e := validateName(req.Name, "policy"); e != nil {
		return nil, e
	}
	if e := validatePolicy(req.Effect, req.Actions); e != nil {
		return nil, e
	}
	id, e := graphID()
	if e != nil {
		return nil, e
	}
	v := &domain.Policy{ID: id, OrgID: org, Name: req.Name, Description: req.Description, Effect: req.Effect, Actions: req.Actions}
	if e = s.graphRepo.CreatePolicy(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) GetPolicy(id string) (*domain.Policy, error) { return s.graphRepo.GetPolicy(id) }
func (s *Service) ListPolicies(org, _ string) ([]*domain.Policy, error) {
	return s.graphRepo.ListPolicies(org, "")
}
func (s *Service) UpdatePolicy(id string, req domain.UpdatePolicyRequest) (*domain.Policy, error) {
	v, e := s.GetPolicy(id)
	if e != nil {
		return nil, e
	}
	if req.Name != nil {
		if e = validateName(*req.Name, "policy"); e != nil {
			return nil, e
		}
		v.Name = *req.Name
	}
	if req.Description != nil {
		v.Description = *req.Description
	}
	if req.Effect != nil {
		v.Effect = *req.Effect
	}
	if req.Actions != nil {
		v.Actions = *req.Actions
	}
	if e = validatePolicy(v.Effect, v.Actions); e != nil {
		return nil, e
	}
	if e = s.graphRepo.UpdatePolicy(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) DeletePolicy(id string) error { return s.graphRepo.DeletePolicy(id) }

func (s *Service) CreateBinding(policy string, req domain.CreatePolicyBindingRequest) (*domain.PolicyBinding, error) {
	parent, e := s.GetPolicy(policy)
	if e != nil {
		return nil, domain.NotFoundError("policy", policy)
	}
	id, e := graphID()
	if e != nil {
		return nil, e
	}
	v := &domain.PolicyBinding{ID: id, OrgID: parent.OrgID, PolicyID: parent.ID, PrincipalType: req.PrincipalType, PrincipalID: req.PrincipalID, TargetType: req.TargetType, TargetID: req.TargetID}
	if e = s.graphRepo.CreateBinding(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) GetBinding(id string) (*domain.PolicyBinding, error) {
	return s.graphRepo.GetBinding(id)
}
func (s *Service) ListBindings(policy string) ([]*domain.PolicyBinding, error) {
	return s.graphRepo.ListBindings(policy)
}
func (s *Service) DeleteBinding(id string) error { return s.graphRepo.DeleteBinding(id) }

func validateLB(protocol string, port int, algorithm, path string) error {
	if protocol != "http" && protocol != "tcp" {
		return domain.InvalidInputError("protocol must be http or tcp", nil)
	}
	if e := validatePort(port); e != nil {
		return e
	}
	if algorithm != "round_robin" && algorithm != "least_connections" {
		return domain.InvalidInputError("invalid load balancer algorithm", nil)
	}
	if protocol == "http" && !strings.HasPrefix(path, "/") {
		return domain.InvalidInputError("HTTP health_check_path must start with /", nil)
	}
	if protocol == "tcp" && path != "" {
		return domain.InvalidInputError("TCP health_check_path must be empty", nil)
	}
	return nil
}
func (s *Service) CreateLoadBalancer(project string, req domain.CreateLoadBalancerRequest) (*domain.LoadBalancer, error) {
	if _, e := s.validateProject(project); e != nil {
		return nil, e
	}
	if e := validateName(req.Name, "load balancer"); e != nil {
		return nil, e
	}
	subnet, e := s.GetSubnet(req.SubnetID)
	if e != nil || subnet.ProjectID != project {
		return nil, domain.NotFoundError("subnet", req.SubnetID)
	}
	network, e := s.GetNetwork(subnet.NetworkID)
	if e != nil {
		return nil, e
	}
	req.Protocol = strings.ToLower(req.Protocol)
	if e = validateLB(req.Protocol, req.Port, req.Algorithm, req.HealthCheckPath); e != nil {
		return nil, e
	}
	id, e := graphID()
	if e != nil {
		return nil, e
	}
	v := &domain.LoadBalancer{ID: id, ProjectID: project, Name: req.Name, SubnetID: subnet.ID, Region: network.Region, Protocol: req.Protocol, Port: req.Port, Algorithm: req.Algorithm, HealthCheckPath: req.HealthCheckPath, Status: "active"}
	if e = s.graphRepo.CreateLoadBalancer(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) GetLoadBalancer(id string) (*domain.LoadBalancer, error) {
	return s.graphRepo.GetLoadBalancer(id)
}
func (s *Service) ListLoadBalancers(project string) ([]*domain.LoadBalancer, error) {
	return s.graphRepo.ListLoadBalancers(project)
}
func (s *Service) UpdateLoadBalancer(id string, req domain.UpdateLoadBalancerRequest) (*domain.LoadBalancer, error) {
	v, e := s.GetLoadBalancer(id)
	if e != nil {
		return nil, e
	}
	if req.Name != nil {
		if e = validateName(*req.Name, "load balancer"); e != nil {
			return nil, e
		}
		v.Name = *req.Name
	}
	if req.Algorithm != nil {
		v.Algorithm = *req.Algorithm
	}
	if req.HealthCheckPath != nil {
		v.HealthCheckPath = *req.HealthCheckPath
	}
	if e = validateLB(v.Protocol, v.Port, v.Algorithm, v.HealthCheckPath); e != nil {
		return nil, e
	}
	if e = s.graphRepo.UpdateLoadBalancer(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) DeleteLoadBalancer(id string) error { return s.graphRepo.DeleteLoadBalancer(id) }

func (s *Service) CreateBackend(lb *domain.LoadBalancer, req domain.CreateLoadBalancerBackendRequest) (*domain.LoadBalancerBackend, error) {
	parent, e := s.GetLoadBalancer(lb.ID)
	if e != nil {
		return nil, domain.NotFoundError("load balancer", lb.ID)
	}
	instance, e := s.instanceRepo.GetByID(req.InstanceID)
	if e != nil || instance.ProjectID != parent.ProjectID || instance.SubnetID == nil || *instance.SubnetID != parent.SubnetID {
		return nil, domain.NotFoundError("instance", req.InstanceID)
	}
	if e = validatePort(req.Port); e != nil {
		return nil, e
	}
	if req.Weight < 1 || req.Weight > 100 {
		return nil, domain.InvalidInputError("weight must be between 1 and 100", nil)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id, e := graphID()
	if e != nil {
		return nil, e
	}
	v := &domain.LoadBalancerBackend{ID: id, LoadBalancerID: parent.ID, InstanceID: req.InstanceID, Port: req.Port, Weight: req.Weight, Enabled: enabled, Healthy: enabled && instance.Status == domain.StatusRunning}
	if e = s.graphRepo.CreateBackend(v); e != nil {
		return nil, e
	}
	return v, nil
}
func (s *Service) GetBackend(id string) (*domain.LoadBalancerBackend, error) {
	return s.graphRepo.GetBackend(id)
}
func (s *Service) ListBackends(lb string) ([]*domain.LoadBalancerBackend, error) {
	return s.graphRepo.ListBackends(lb)
}
func (s *Service) UpdateBackend(id string, req domain.UpdateLoadBalancerBackendRequest) (*domain.LoadBalancerBackend, error) {
	v, e := s.GetBackend(id)
	if e != nil {
		return nil, e
	}
	if req.Port != nil {
		if e = validatePort(*req.Port); e != nil {
			return nil, e
		}
		v.Port = *req.Port
	}
	if req.Weight != nil {
		if *req.Weight < 1 || *req.Weight > 100 {
			return nil, domain.InvalidInputError("weight must be between 1 and 100", nil)
		}
		v.Weight = *req.Weight
	}
	if req.Enabled != nil {
		v.Enabled = *req.Enabled
	}
	if e = s.graphRepo.UpdateBackend(v); e != nil {
		return nil, e
	}
	return s.GetBackend(id)
}
func (s *Service) DeleteBackend(id string) error { return s.graphRepo.DeleteBackend(id) }

func (s *Service) EvaluatePolicy(org string, req domain.PolicyEvaluationRequest) (*domain.PolicyEvaluation, error) {
	if req.Action == "" {
		return nil, domain.InvalidInputError("action is required", nil)
	}
	switch req.PrincipalType {
	case "organization":
		if req.PrincipalID != org {
			return nil, domain.NotFoundError("organization", req.PrincipalID)
		}
	case "api_key":
		key, err := s.apiKeyRepo.GetByID(req.PrincipalID)
		if err != nil || key.OrgID != org {
			return nil, domain.NotFoundError("api_key", req.PrincipalID)
		}
	default:
		return nil, domain.InvalidInputError("invalid principal_type", nil)
	}
	targetProject, e := s.evaluationTargetProject(org, req.TargetType, req.TargetID)
	if e != nil {
		return nil, e
	}
	bindings, e := s.graphRepo.ListOrgBindings(org)
	if e != nil {
		return nil, e
	}
	policies, e := s.graphRepo.ListPolicies(org, "")
	if e != nil {
		return nil, e
	}
	policyByID := map[string]*domain.Policy{}
	for _, p := range policies {
		policyByID[p.ID] = p
	}
	result := &domain.PolicyEvaluation{Result: "not_applicable", MatchingBindingIDs: []string{}}
	allow := false
	for _, b := range bindings {
		if b.PrincipalType != req.PrincipalType || b.PrincipalID != req.PrincipalID {
			continue
		}
		p := policyByID[b.PolicyID]
		if p == nil || (!slices.Contains(p.Actions, req.Action) && !slices.Contains(p.Actions, "*")) {
			continue
		}
		matches := b.TargetType == req.TargetType && b.TargetID == req.TargetID
		if b.TargetType == "organization" && b.TargetID == org {
			matches = true
		}
		if b.TargetType == "project" && targetProject != "" && b.TargetID == targetProject {
			matches = true
		}
		if !matches {
			continue
		}
		result.MatchingBindingIDs = append(result.MatchingBindingIDs, b.ID)
		if p.Effect == "deny" {
			result.Result = "deny"
		} else {
			allow = true
		}
	}
	if result.Result != "deny" && allow {
		result.Result = "allow"
	}
	return result, nil
}
func (s *Service) evaluationTargetProject(org, kind, id string) (string, error) {
	switch kind {
	case "organization":
		if id != org {
			return "", domain.NotFoundError(kind, id)
		}
		return "", nil
	case "project":
		p, e := s.projectRepo.GetByID(id)
		if e != nil || p.OrgID != org {
			return "", domain.NotFoundError(kind, id)
		}
		return p.ID, nil
	case "network":
		v, e := s.GetNetwork(id)
		if e != nil {
			return "", domain.NotFoundError(kind, id)
		}
		return s.projectInOrg(org, v.ProjectID, kind, id)
	case "subnet":
		v, e := s.GetSubnet(id)
		if e != nil {
			return "", domain.NotFoundError(kind, id)
		}
		return s.projectInOrg(org, v.ProjectID, kind, id)
	case "instance":
		v, e := s.instanceRepo.GetByID(id)
		if e != nil {
			return "", domain.NotFoundError(kind, id)
		}
		return s.projectInOrg(org, v.ProjectID, kind, id)
	case "disk":
		v, e := s.GetDisk(id)
		if e != nil {
			return "", domain.NotFoundError(kind, id)
		}
		return s.projectInOrg(org, v.ProjectID, kind, id)
	case "load_balancer":
		v, e := s.GetLoadBalancer(id)
		if e != nil {
			return "", domain.NotFoundError(kind, id)
		}
		return s.projectInOrg(org, v.ProjectID, kind, id)
	case "bucket":
		v, e := s.bucketRepo.GetByID(id)
		if e != nil {
			return "", domain.NotFoundError(kind, id)
		}
		return s.projectInOrg(org, v.ProjectID, kind, id)
	default:
		return "", domain.InvalidInputError("invalid target_type", nil)
	}
}
func (s *Service) projectInOrg(org, project, kind, id string) (string, error) {
	p, e := s.projectRepo.GetByID(project)
	if e != nil || p.OrgID != org {
		return "", domain.NotFoundError(kind, id)
	}
	return project, nil
}
