package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
)

func (h *Handler) graphCollection(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		org, err := h.resolveOrg(r)
		if err != nil {
			h.writeError(w, err)
			return
		}
		var project *domain.Project
		if mux.Vars(r)["project"] != "" {
			project, err = h.resolveProject(r)
			if err != nil {
				h.writeError(w, err)
				return
			}
		}
		if r.Method == http.MethodGet {
			var result any
			switch kind {
			case "network":
				result, err = h.service.ListNetworks(project.ID)
			case "subnet":
				var parent *domain.Network
				parent, err = h.scopedNetwork(r, project)
				if err == nil {
					result, err = h.service.ListSubnets(parent.ID)
				}
			case "disk":
				result, err = h.service.ListDisks(project.ID)
			case "attachment":
				var parent *domain.Disk
				parent, err = h.scopedDisk(r, project)
				if err == nil {
					result, err = h.service.ListAttachments(parent.ID)
				}
			case "policy":
				result, err = h.service.ListPolicies(org.ID, projectID(project))
			case "binding":
				var parent *domain.Policy
				parent, err = h.scopedPolicy(r, org, project)
				if err == nil {
					result, err = h.service.ListBindings(parent.ID)
				}
			case "load-balancer":
				result, err = h.service.ListLoadBalancers(project.ID)
			case "backend":
				var parent *domain.LoadBalancer
				parent, err = h.scopedLoadBalancer(r, project)
				if err == nil {
					result, err = h.service.ListBackends(parent.ID)
				}
			}
			if err != nil {
				h.writeError(w, err)
				return
			}
			h.writeJSON(w, http.StatusOK, result)
			return
		}
		var result any
		switch kind {
		case "network":
			var req domain.CreateNetworkRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.CreateNetwork(project.ID, req)
			}
		case "subnet":
			var req domain.CreateSubnetRequest
			var parent *domain.Network
			if parent, err = h.scopedNetwork(r, project); err == nil {
				if err = h.decodeJSON(r, &req); err == nil {
					result, err = h.service.CreateSubnet(parent, req)
				}
			}
		case "disk":
			var req domain.CreateDiskRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.CreateDisk(project.ID, req)
			}
		case "attachment":
			var req domain.CreateDiskAttachmentRequest
			var parent *domain.Disk
			if parent, err = h.scopedDisk(r, project); err == nil {
				if err = h.decodeJSON(r, &req); err == nil {
					result, err = h.service.CreateAttachment(parent, req)
				}
			}
		case "policy":
			var req domain.CreatePolicyRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.CreatePolicy(org.ID, projectID(project), req)
			}
		case "binding":
			var req domain.CreatePolicyBindingRequest
			var parent *domain.Policy
			if parent, err = h.scopedPolicy(r, org, project); err == nil {
				if err = h.decodeJSON(r, &req); err == nil {
					result, err = h.service.CreateBinding(parent.ID, req)
				}
			}
		case "load-balancer":
			var req domain.CreateLoadBalancerRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.CreateLoadBalancer(project.ID, req)
			}
		case "backend":
			var req domain.CreateLoadBalancerBackendRequest
			var parent *domain.LoadBalancer
			if parent, err = h.scopedLoadBalancer(r, project); err == nil {
				if err = h.decodeJSON(r, &req); err == nil {
					result, err = h.service.CreateBackend(parent, req)
				}
			}
		}
		if err != nil {
			h.writeError(w, err)
			return
		}
		h.writeJSON(w, http.StatusCreated, result)
	}
}

func (h *Handler) graphItem(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		org, err := h.resolveOrg(r)
		if err != nil {
			h.writeError(w, err)
			return
		}
		var project *domain.Project
		if mux.Vars(r)["project"] != "" {
			project, err = h.resolveProject(r)
			if err != nil {
				h.writeError(w, err)
				return
			}
		}
		id := mux.Vars(r)["id"]
		var current any
		switch kind {
		case "network":
			current, err = h.service.GetNetwork(id)
			if err == nil && current.(*domain.Network).ProjectID != project.ID {
				err = domain.NotFoundError("network", id)
			}
		case "subnet":
			var parent *domain.Network
			parent, err = h.scopedNetwork(r, project)
			if err == nil {
				current, err = h.service.GetSubnet(id)
				if err == nil && current.(*domain.Subnet).NetworkID != parent.ID {
					err = domain.NotFoundError("subnet", id)
				}
			}
		case "disk":
			current, err = h.service.GetDisk(id)
			if err == nil && current.(*domain.Disk).ProjectID != project.ID {
				err = domain.NotFoundError("disk", id)
			}
		case "attachment":
			var parent *domain.Disk
			parent, err = h.scopedDisk(r, project)
			if err == nil {
				current, err = h.service.GetAttachment(id)
				if err == nil && current.(*domain.DiskAttachment).DiskID != parent.ID {
					err = domain.NotFoundError("disk attachment", id)
				}
			}
		case "policy":
			current, err = h.service.GetPolicy(id)
			if err == nil && !policyMatches(current.(*domain.Policy), org, project) {
				err = domain.NotFoundError("policy", id)
			}
		case "binding":
			var parent *domain.Policy
			parent, err = h.scopedPolicy(r, org, project)
			if err == nil {
				current, err = h.service.GetBinding(id)
				if err == nil && current.(*domain.PolicyBinding).PolicyID != parent.ID {
					err = domain.NotFoundError("policy binding", id)
				}
			}
		case "load-balancer":
			current, err = h.service.GetLoadBalancer(id)
			if err == nil && current.(*domain.LoadBalancer).ProjectID != project.ID {
				err = domain.NotFoundError("load balancer", id)
			}
		case "backend":
			var parent *domain.LoadBalancer
			parent, err = h.scopedLoadBalancer(r, project)
			if err == nil {
				current, err = h.service.GetBackend(id)
				if err == nil && current.(*domain.LoadBalancerBackend).LoadBalancerID != parent.ID {
					err = domain.NotFoundError("load balancer backend", id)
				}
			}
		}
		if err != nil {
			h.writeError(w, err)
			return
		}
		if r.Method == http.MethodGet {
			h.writeJSON(w, http.StatusOK, current)
			return
		}
		if r.Method == http.MethodDelete {
			switch kind {
			case "network":
				err = h.service.DeleteNetwork(id)
			case "subnet":
				err = h.service.DeleteSubnet(id)
			case "disk":
				err = h.service.DeleteDisk(id)
			case "attachment":
				err = h.service.DeleteAttachment(id)
			case "policy":
				err = h.service.DeletePolicy(id)
			case "binding":
				err = h.service.DeleteBinding(id)
			case "load-balancer":
				err = h.service.DeleteLoadBalancer(id)
			case "backend":
				err = h.service.DeleteBackend(id)
			}
			if err != nil {
				h.writeError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var result any
		switch kind {
		case "network":
			var req domain.UpdateNetworkRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.UpdateNetwork(id, req)
			}
		case "subnet":
			var req domain.UpdateSubnetRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.UpdateSubnet(id, req)
			}
		case "disk":
			var req domain.UpdateDiskRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.UpdateDisk(id, req)
			}
		case "attachment":
			var req domain.UpdateDiskAttachmentRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.UpdateAttachment(id, req)
			}
		case "policy":
			var req domain.UpdatePolicyRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.UpdatePolicy(id, req)
			}
		case "binding":
			var req domain.UpdatePolicyBindingRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.UpdateBinding(id, req)
			}
		case "load-balancer":
			var req domain.UpdateLoadBalancerRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.UpdateLoadBalancer(id, req)
			}
		case "backend":
			var req domain.UpdateLoadBalancerBackendRequest
			if err = h.decodeJSON(r, &req); err == nil {
				result, err = h.service.UpdateBackend(id, req)
			}
		}
		if err != nil {
			h.writeError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, result)
	}
}

func projectID(p *domain.Project) string {
	if p == nil {
		return ""
	}
	return p.ID
}
func policyMatches(v *domain.Policy, org *domain.Organization, project *domain.Project) bool {
	return v.OrgID == org.ID && v.ProjectID == projectID(project)
}
func (h *Handler) scopedNetwork(r *http.Request, p *domain.Project) (*domain.Network, error) {
	id := mux.Vars(r)["network_id"]
	v, err := h.service.GetNetwork(id)
	if err == nil && v.ProjectID != p.ID {
		err = domain.NotFoundError("network", id)
	}
	return v, err
}
func (h *Handler) scopedDisk(r *http.Request, p *domain.Project) (*domain.Disk, error) {
	id := mux.Vars(r)["disk_id"]
	v, err := h.service.GetDisk(id)
	if err == nil && v.ProjectID != p.ID {
		err = domain.NotFoundError("disk", id)
	}
	return v, err
}
func (h *Handler) scopedLoadBalancer(r *http.Request, p *domain.Project) (*domain.LoadBalancer, error) {
	id := mux.Vars(r)["load_balancer_id"]
	v, err := h.service.GetLoadBalancer(id)
	if err == nil && v.ProjectID != p.ID {
		err = domain.NotFoundError("load balancer", id)
	}
	return v, err
}
func (h *Handler) scopedPolicy(r *http.Request, o *domain.Organization, p *domain.Project) (*domain.Policy, error) {
	id := mux.Vars(r)["policy_id"]
	v, err := h.service.GetPolicy(id)
	if err == nil && !policyMatches(v, o, p) {
		err = domain.NotFoundError("policy", id)
	}
	return v, err
}
