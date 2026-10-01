package service

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/hypertf/nahcloud/domain"
)

func graphID() (string, error) { return generateID() }

func validateGraphName(name, resource string) error { return validateName(name, resource) }
func validateCIDR(cidr string) error {
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return domain.InvalidInputError("invalid CIDR", map[string]interface{}{"cidr": cidr})
	}
	return nil
}
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return domain.InvalidInputError("port must be between 1 and 65535", nil)
	}
	return nil
}
func (s *Service) validateProject(projectID string) (*domain.Project, error) {
	project, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ForeignKeyViolationError("project", "id", projectID)
		}
		return nil, err
	}
	return project, nil
}
func (s *Service) validateInstanceProject(id, projectID string) error {
	v, err := s.instanceRepo.GetByID(id)
	if err != nil {
		return domain.ForeignKeyViolationError("instance", "id", id)
	}
	if v.ProjectID != projectID {
		return domain.ForeignKeyViolationError("instance", "id", id)
	}
	return nil
}

func (s *Service) CreateNetwork(projectID string, req domain.CreateNetworkRequest) (*domain.Network, error) {
	if _, err := s.validateProject(projectID); err != nil {
		return nil, err
	}
	if err := validateGraphName(req.Name, "network"); err != nil {
		return nil, err
	}
	if err := validateCIDR(req.CIDR); err != nil {
		return nil, err
	}
	id, err := graphID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}
	v := &domain.Network{ID: id, ProjectID: projectID, Name: req.Name, CIDR: req.CIDR}
	if err = s.graphRepo.CreateNetwork(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetNetwork(id string) (*domain.Network, error) { return s.graphRepo.GetNetwork(id) }
func (s *Service) ListNetworks(projectID string) ([]*domain.Network, error) {
	return s.graphRepo.ListNetworks(projectID)
}
func (s *Service) UpdateNetwork(id string, req domain.UpdateNetworkRequest) (*domain.Network, error) {
	v, err := s.GetNetwork(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if err = validateGraphName(*req.Name, "network"); err != nil {
			return nil, err
		}
		v.Name = *req.Name
	}
	if req.CIDR != nil {
		if err = validateCIDR(*req.CIDR); err != nil {
			return nil, err
		}
		_, candidate, _ := net.ParseCIDR(*req.CIDR)
		subnets, listErr := s.graphRepo.ListSubnets(v.ID)
		if listErr != nil {
			return nil, listErr
		}
		for _, subnet := range subnets {
			subnetIP, subnetCIDR, _ := net.ParseCIDR(subnet.CIDR)
			candidatePrefix, candidateBits := candidate.Mask.Size()
			subnetPrefix, subnetBits := subnetCIDR.Mask.Size()
			if candidateBits != subnetBits || !candidate.Contains(subnetIP) || subnetPrefix < candidatePrefix {
				return nil, domain.InvalidInputError("network CIDR must contain existing subnets", nil)
			}
		}
		v.CIDR = *req.CIDR
	}
	if err = s.graphRepo.UpdateNetwork(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) DeleteNetwork(id string) error { return s.graphRepo.DeleteNetwork(id) }

func (s *Service) CreateSubnet(network *domain.Network, req domain.CreateSubnetRequest) (*domain.Subnet, error) {
	storedNetwork, err := s.graphRepo.GetNetwork(network.ID)
	if err != nil {
		return nil, domain.ForeignKeyViolationError("network", "id", network.ID)
	}
	if err := validateGraphName(req.Name, "subnet"); err != nil {
		return nil, err
	}
	if err := validateCIDR(req.CIDR); err != nil {
		return nil, err
	}
	_, parentCIDR, _ := net.ParseCIDR(storedNetwork.CIDR)
	subnetIP, subnetCIDR, _ := net.ParseCIDR(req.CIDR)
	parentPrefix, _ := parentCIDR.Mask.Size()
	subnetPrefix, _ := subnetCIDR.Mask.Size()
	if !parentCIDR.Contains(subnetIP) || subnetPrefix < parentPrefix {
		return nil, domain.InvalidInputError("subnet CIDR must be contained by network CIDR", nil)
	}
	id, err := graphID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}
	v := &domain.Subnet{ID: id, NetworkID: storedNetwork.ID, ProjectID: storedNetwork.ProjectID, Name: req.Name, CIDR: req.CIDR}
	if err = s.graphRepo.CreateSubnet(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetSubnet(id string) (*domain.Subnet, error) { return s.graphRepo.GetSubnet(id) }
func (s *Service) ListSubnets(networkID string) ([]*domain.Subnet, error) {
	return s.graphRepo.ListSubnets(networkID)
}
func (s *Service) UpdateSubnet(id string, req domain.UpdateSubnetRequest) (*domain.Subnet, error) {
	v, err := s.GetSubnet(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if err = validateGraphName(*req.Name, "subnet"); err != nil {
			return nil, err
		}
		v.Name = *req.Name
	}
	if req.CIDR != nil {
		if err = validateCIDR(*req.CIDR); err != nil {
			return nil, err
		}
		network, parentErr := s.graphRepo.GetNetwork(v.NetworkID)
		if parentErr != nil {
			return nil, parentErr
		}
		_, parentCIDR, _ := net.ParseCIDR(network.CIDR)
		subnetIP, subnetCIDR, _ := net.ParseCIDR(*req.CIDR)
		parentPrefix, _ := parentCIDR.Mask.Size()
		subnetPrefix, _ := subnetCIDR.Mask.Size()
		if !parentCIDR.Contains(subnetIP) || subnetPrefix < parentPrefix {
			return nil, domain.InvalidInputError("subnet CIDR must be contained by network CIDR", nil)
		}
		v.CIDR = *req.CIDR
	}
	if err = s.graphRepo.UpdateSubnet(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) DeleteSubnet(id string) error { return s.graphRepo.DeleteSubnet(id) }

func (s *Service) CreateDisk(projectID string, req domain.CreateDiskRequest) (*domain.Disk, error) {
	if _, err := s.validateProject(projectID); err != nil {
		return nil, err
	}
	if err := validateGraphName(req.Name, "disk"); err != nil {
		return nil, err
	}
	if req.SizeGB < 1 {
		return nil, domain.InvalidInputError("size_gb must be positive", nil)
	}
	id, err := graphID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}
	v := &domain.Disk{ID: id, ProjectID: projectID, Name: req.Name, SizeGB: req.SizeGB}
	if err = s.graphRepo.CreateDisk(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetDisk(id string) (*domain.Disk, error) { return s.graphRepo.GetDisk(id) }
func (s *Service) ListDisks(projectID string) ([]*domain.Disk, error) {
	return s.graphRepo.ListDisks(projectID)
}
func (s *Service) UpdateDisk(id string, req domain.UpdateDiskRequest) (*domain.Disk, error) {
	v, err := s.GetDisk(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if err = validateGraphName(*req.Name, "disk"); err != nil {
			return nil, err
		}
		v.Name = *req.Name
	}
	if req.SizeGB != nil {
		if *req.SizeGB < v.SizeGB {
			return nil, domain.InvalidInputError("disk size cannot be decreased", nil)
		}
		v.SizeGB = *req.SizeGB
	}
	if err = s.graphRepo.UpdateDisk(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) DeleteDisk(id string) error { return s.graphRepo.DeleteDisk(id) }

func (s *Service) CreateAttachment(disk *domain.Disk, req domain.CreateDiskAttachmentRequest) (*domain.DiskAttachment, error) {
	storedDisk, err := s.graphRepo.GetDisk(disk.ID)
	if err != nil {
		return nil, domain.ForeignKeyViolationError("disk", "id", disk.ID)
	}
	if req.Device == "" {
		return nil, domain.InvalidInputError("device cannot be empty", nil)
	}
	if err := s.validateInstanceProject(req.InstanceID, storedDisk.ProjectID); err != nil {
		return nil, err
	}
	id, err := graphID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}
	v := &domain.DiskAttachment{ID: id, DiskID: storedDisk.ID, ProjectID: storedDisk.ProjectID, InstanceID: req.InstanceID, Device: req.Device}
	if err = s.graphRepo.CreateAttachment(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetAttachment(id string) (*domain.DiskAttachment, error) {
	return s.graphRepo.GetAttachment(id)
}
func (s *Service) ListAttachments(diskID string) ([]*domain.DiskAttachment, error) {
	return s.graphRepo.ListAttachments(diskID)
}
func (s *Service) UpdateAttachment(id string, req domain.UpdateDiskAttachmentRequest) (*domain.DiskAttachment, error) {
	v, err := s.GetAttachment(id)
	if err != nil {
		return nil, err
	}
	if req.Device != nil {
		if *req.Device == "" {
			return nil, domain.InvalidInputError("device cannot be empty", nil)
		}
		v.Device = *req.Device
	}
	if err = s.graphRepo.UpdateAttachment(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) DeleteAttachment(id string) error { return s.graphRepo.DeleteAttachment(id) }

func (s *Service) CreatePolicy(orgID, projectID string, req domain.CreatePolicyRequest) (*domain.Policy, error) {
	if _, err := s.orgRepo.GetByID(orgID); err != nil {
		return nil, domain.ForeignKeyViolationError("organization", "id", orgID)
	}
	if projectID != "" {
		project, err := s.validateProject(projectID)
		if err != nil {
			return nil, err
		}
		if project.OrgID != orgID {
			return nil, domain.ForeignKeyViolationError("project", "id", projectID)
		}
	}
	if err := validateGraphName(req.Name, "policy"); err != nil {
		return nil, err
	}
	if !json.Valid([]byte(req.Document)) {
		return nil, domain.InvalidInputError("policy document must be valid JSON", nil)
	}
	id, err := graphID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}
	v := &domain.Policy{ID: id, OrgID: orgID, ProjectID: projectID, Name: req.Name, Document: req.Document}
	if err = s.graphRepo.CreatePolicy(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetPolicy(id string) (*domain.Policy, error) { return s.graphRepo.GetPolicy(id) }
func (s *Service) ListPolicies(orgID, projectID string) ([]*domain.Policy, error) {
	return s.graphRepo.ListPolicies(orgID, projectID)
}
func (s *Service) UpdatePolicy(id string, req domain.UpdatePolicyRequest) (*domain.Policy, error) {
	v, err := s.GetPolicy(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if err = validateGraphName(*req.Name, "policy"); err != nil {
			return nil, err
		}
		v.Name = *req.Name
	}
	if req.Document != nil {
		if !json.Valid([]byte(*req.Document)) {
			return nil, domain.InvalidInputError("policy document must be valid JSON", nil)
		}
		v.Document = *req.Document
	}
	if err = s.graphRepo.UpdatePolicy(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) DeletePolicy(id string) error { return s.graphRepo.DeletePolicy(id) }

func (s *Service) CreateBinding(policyID string, req domain.CreatePolicyBindingRequest) (*domain.PolicyBinding, error) {
	if _, err := s.graphRepo.GetPolicy(policyID); err != nil {
		return nil, domain.ForeignKeyViolationError("policy", "id", policyID)
	}
	if req.Principal == "" || req.Role == "" {
		return nil, domain.InvalidInputError("principal and role are required", nil)
	}
	id, err := graphID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}
	v := &domain.PolicyBinding{ID: id, PolicyID: policyID, Principal: req.Principal, Role: req.Role}
	if err = s.graphRepo.CreateBinding(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetBinding(id string) (*domain.PolicyBinding, error) {
	return s.graphRepo.GetBinding(id)
}
func (s *Service) ListBindings(policyID string) ([]*domain.PolicyBinding, error) {
	return s.graphRepo.ListBindings(policyID)
}
func (s *Service) UpdateBinding(id string, req domain.UpdatePolicyBindingRequest) (*domain.PolicyBinding, error) {
	v, err := s.GetBinding(id)
	if err != nil {
		return nil, err
	}
	if req.Principal != nil {
		v.Principal = *req.Principal
	}
	if req.Role != nil {
		v.Role = *req.Role
	}
	if v.Principal == "" || v.Role == "" {
		return nil, domain.InvalidInputError("principal and role are required", nil)
	}
	if err = s.graphRepo.UpdateBinding(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) DeleteBinding(id string) error { return s.graphRepo.DeleteBinding(id) }

func (s *Service) CreateLoadBalancer(projectID string, req domain.CreateLoadBalancerRequest) (*domain.LoadBalancer, error) {
	if _, err := s.validateProject(projectID); err != nil {
		return nil, err
	}
	if err := validateGraphName(req.Name, "load balancer"); err != nil {
		return nil, err
	}
	req.Protocol = strings.ToLower(req.Protocol)
	if req.Protocol != "tcp" && req.Protocol != "http" && req.Protocol != "https" {
		return nil, domain.InvalidInputError("protocol must be tcp, http, or https", nil)
	}
	if err := validatePort(req.Port); err != nil {
		return nil, err
	}
	id, err := graphID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}
	v := &domain.LoadBalancer{ID: id, ProjectID: projectID, Name: req.Name, Protocol: req.Protocol, Port: req.Port}
	if err = s.graphRepo.CreateLoadBalancer(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetLoadBalancer(id string) (*domain.LoadBalancer, error) {
	return s.graphRepo.GetLoadBalancer(id)
}
func (s *Service) ListLoadBalancers(projectID string) ([]*domain.LoadBalancer, error) {
	return s.graphRepo.ListLoadBalancers(projectID)
}
func (s *Service) UpdateLoadBalancer(id string, req domain.UpdateLoadBalancerRequest) (*domain.LoadBalancer, error) {
	v, err := s.GetLoadBalancer(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if err = validateGraphName(*req.Name, "load balancer"); err != nil {
			return nil, err
		}
		v.Name = *req.Name
	}
	if req.Protocol != nil {
		p := strings.ToLower(*req.Protocol)
		if p != "tcp" && p != "http" && p != "https" {
			return nil, domain.InvalidInputError("protocol must be tcp, http, or https", nil)
		}
		v.Protocol = p
	}
	if req.Port != nil {
		if err = validatePort(*req.Port); err != nil {
			return nil, err
		}
		v.Port = *req.Port
	}
	if err = s.graphRepo.UpdateLoadBalancer(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) DeleteLoadBalancer(id string) error { return s.graphRepo.DeleteLoadBalancer(id) }

func (s *Service) CreateBackend(lb *domain.LoadBalancer, req domain.CreateLoadBalancerBackendRequest) (*domain.LoadBalancerBackend, error) {
	storedLB, err := s.graphRepo.GetLoadBalancer(lb.ID)
	if err != nil {
		return nil, domain.ForeignKeyViolationError("load balancer", "id", lb.ID)
	}
	if err := s.validateInstanceProject(req.InstanceID, storedLB.ProjectID); err != nil {
		return nil, err
	}
	if err := validatePort(req.Port); err != nil {
		return nil, err
	}
	if req.Weight == 0 {
		req.Weight = 1
	}
	if req.Weight < 1 || req.Weight > 100 {
		return nil, domain.InvalidInputError("weight must be between 1 and 100", nil)
	}
	id, err := graphID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}
	v := &domain.LoadBalancerBackend{ID: id, LoadBalancerID: storedLB.ID, ProjectID: storedLB.ProjectID, InstanceID: req.InstanceID, Port: req.Port, Weight: req.Weight}
	if err = s.graphRepo.CreateBackend(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) GetBackend(id string) (*domain.LoadBalancerBackend, error) {
	return s.graphRepo.GetBackend(id)
}
func (s *Service) ListBackends(lbID string) ([]*domain.LoadBalancerBackend, error) {
	return s.graphRepo.ListBackends(lbID)
}
func (s *Service) UpdateBackend(id string, req domain.UpdateLoadBalancerBackendRequest) (*domain.LoadBalancerBackend, error) {
	v, err := s.GetBackend(id)
	if err != nil {
		return nil, err
	}
	if req.Port != nil {
		if err = validatePort(*req.Port); err != nil {
			return nil, err
		}
		v.Port = *req.Port
	}
	if req.Weight != nil {
		if *req.Weight < 1 || *req.Weight > 100 {
			return nil, domain.InvalidInputError("weight must be between 1 and 100", nil)
		}
		v.Weight = *req.Weight
	}
	if err = s.graphRepo.UpdateBackend(v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s *Service) DeleteBackend(id string) error { return s.graphRepo.DeleteBackend(id) }
