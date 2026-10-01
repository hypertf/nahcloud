package web

import (
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/service"
	"github.com/hypertf/nahcloud/web/static"
)

// GraphScope is resolved from the authenticated organization and project slug
// before the adapter is called. Policies use OrgID; project resources use
// ProjectID. Adapters must return not found for references outside this scope.
type GraphScope struct {
	OrgID     string
	ProjectID string
}

// GraphConsole is the web-owned integration boundary for the frozen graph
// service contract. It keeps backend domain types out of templates while PR #4
// is corrected and permits a direct service adapter once those types land.
type GraphConsole interface {
	Snapshot(scope GraphScope) (GraphSnapshot, error)
	Create(scope GraphScope, input GraphCreateInput) error
	Delete(scope GraphScope, kind, id string) error
}

type serviceGraphConsole struct {
	service *service.Service
}

func newServiceGraphConsole(svc *service.Service) GraphConsole {
	return &serviceGraphConsole{service: svc}
}

func (a *serviceGraphConsole) validateScope(scope GraphScope) error {
	project, err := a.service.GetProject(scope.ProjectID)
	if err != nil || project.OrgID != scope.OrgID {
		return domain.NotFoundError("project", scope.ProjectID)
	}
	return nil
}

func (a *serviceGraphConsole) Snapshot(scope GraphScope) (GraphSnapshot, error) {
	if err := a.validateScope(scope); err != nil {
		return GraphSnapshot{}, err
	}
	instances, err := a.service.ListInstances(domain.InstanceListOptions{ProjectID: scope.ProjectID})
	if err != nil {
		return GraphSnapshot{}, err
	}
	instanceByID := make(map[string]*domain.Instance, len(instances))
	snapshot := GraphSnapshot{Instances: make([]GraphInstance, 0, len(instances))}
	instanceViewByID := make(map[string]*GraphInstance, len(instances))
	for _, instance := range instances {
		instanceByID[instance.ID] = instance
		view := GraphInstance{ID: instance.ID, Name: instance.Name, Region: instance.Region, Status: instance.Status}
		if instance.SubnetID != nil {
			view.SubnetID = *instance.SubnetID
		}
		snapshot.Instances = append(snapshot.Instances, view)
		instanceViewByID[instance.ID] = &snapshot.Instances[len(snapshot.Instances)-1]
	}

	networks, err := a.service.ListNetworks(scope.ProjectID)
	if err != nil {
		return GraphSnapshot{}, err
	}
	subnetByID := map[string]*GraphSubnet{}
	for _, network := range networks {
		subnets, err := a.service.ListSubnets(network.ID)
		if err != nil {
			return GraphSnapshot{}, err
		}
		view := GraphNetwork{ID: network.ID, Name: network.Name, Region: network.Region, Subnets: make([]GraphSubnet, 0, len(subnets))}
		for _, subnet := range subnets {
			view.Subnets = append(view.Subnets, GraphSubnet{ID: subnet.ID, Name: subnet.Name, CIDR: subnet.CIDR})
			subnetByID[subnet.ID] = &view.Subnets[len(view.Subnets)-1]
		}
		snapshot.Networks = append(snapshot.Networks, view)
	}
	for id, instance := range instanceByID {
		if instance.SubnetID != nil {
			if subnet := subnetByID[*instance.SubnetID]; subnet != nil {
				subnet.InstanceCount++
				instanceViewByID[id].SubnetName = subnet.Name
			}
		}
	}

	disks, err := a.service.ListDisks(scope.ProjectID)
	if err != nil {
		return GraphSnapshot{}, err
	}
	for _, disk := range disks {
		view := GraphDisk{ID: disk.ID, Name: disk.Name, Region: disk.Region, Type: disk.Type, SizeGB: disk.SizeGB}
		attachments, err := a.service.ListAttachments(disk.ID)
		if err != nil {
			return GraphSnapshot{}, err
		}
		if len(attachments) > 0 {
			attachment := attachments[0]
			view.Attachment = &GraphAttachment{ID: attachment.ID, InstanceID: attachment.InstanceID, Device: attachment.Device}
			if instance := instanceByID[attachment.InstanceID]; instance != nil {
				view.Attachment.InstanceName = instance.Name
				instanceViewByID[instance.ID].Disks = append(instanceViewByID[instance.ID].Disks, GraphResourceLink{ID: disk.ID, Name: disk.Name})
			}
		}
		snapshot.Disks = append(snapshot.Disks, view)
	}

	policies, err := a.service.ListPolicies(scope.OrgID, "")
	if err != nil {
		return GraphSnapshot{}, err
	}
	for _, policy := range policies {
		view := GraphPolicy{ID: policy.ID, Name: policy.Name, Description: policy.Description, Effect: policy.Effect, Actions: policy.Actions}
		bindings, err := a.service.ListBindings(policy.ID)
		if err != nil {
			return GraphSnapshot{}, err
		}
		for _, binding := range bindings {
			view.Bindings = append(view.Bindings, GraphBinding{ID: binding.ID, PrincipalType: binding.PrincipalType, PrincipalID: binding.PrincipalID, TargetType: binding.TargetType, TargetID: binding.TargetID, TargetName: a.targetName(binding.TargetType, binding.TargetID)})
		}
		snapshot.Policies = append(snapshot.Policies, view)
	}

	loadBalancers, err := a.service.ListLoadBalancers(scope.ProjectID)
	if err != nil {
		return GraphSnapshot{}, err
	}
	for _, lb := range loadBalancers {
		view := GraphLoadBalancer{ID: lb.ID, Name: lb.Name, SubnetID: lb.SubnetID, Region: lb.Region, Protocol: lb.Protocol, Port: lb.Port, Algorithm: lb.Algorithm, HealthCheckPath: lb.HealthCheckPath, Status: lb.Status}
		if subnet := subnetByID[lb.SubnetID]; subnet != nil {
			view.SubnetName = subnet.Name
			subnet.LBCount++
		}
		backends, err := a.service.ListBackends(lb.ID)
		if err != nil {
			return GraphSnapshot{}, err
		}
		for _, backend := range backends {
			backendView := GraphBackend{ID: backend.ID, InstanceID: backend.InstanceID, Port: backend.Port, Weight: backend.Weight, Enabled: backend.Enabled, Healthy: backend.Healthy}
			if instance := instanceByID[backend.InstanceID]; instance != nil {
				backendView.InstanceName = instance.Name
				instanceViewByID[instance.ID].LoadBalancers = append(instanceViewByID[instance.ID].LoadBalancers, GraphLBMembership{ID: lb.ID, Name: lb.Name, BackendID: backend.ID, Port: backend.Port, Healthy: backend.Healthy})
			}
			view.Backends = append(view.Backends, backendView)
		}
		snapshot.LoadBalancers = append(snapshot.LoadBalancers, view)
	}
	return snapshot, nil
}

func (a *serviceGraphConsole) targetName(kind, id string) string {
	switch kind {
	case "project":
		if value, err := a.service.GetProject(id); err == nil {
			return value.Name
		}
	case "network":
		if value, err := a.service.GetNetwork(id); err == nil {
			return value.Name
		}
	case "subnet":
		if value, err := a.service.GetSubnet(id); err == nil {
			return value.Name
		}
	case "instance":
		if value, err := a.service.GetInstance(id); err == nil {
			return value.Name
		}
	case "disk":
		if value, err := a.service.GetDisk(id); err == nil {
			return value.Name
		}
	case "load_balancer":
		if value, err := a.service.GetLoadBalancer(id); err == nil {
			return value.Name
		}
	case "bucket":
		if value, err := a.service.GetBucket(id); err == nil {
			return value.Name
		}
	case "organization":
		if value, err := a.service.GetOrganization(id); err == nil {
			return value.Name
		}
	}
	return id
}

func (a *serviceGraphConsole) Create(scope GraphScope, input GraphCreateInput) error {
	if err := a.validateScope(scope); err != nil {
		return err
	}
	switch input.Kind {
	case "network":
		_, err := a.service.CreateNetwork(scope.ProjectID, domain.CreateNetworkRequest{Name: input.Name, Region: input.Region})
		return err
	case "subnet":
		parent, err := a.service.GetNetwork(input.ParentID)
		if err != nil || parent.ProjectID != scope.ProjectID {
			return domain.NotFoundError("network", input.ParentID)
		}
		_, err = a.service.CreateSubnet(parent, domain.CreateSubnetRequest{Name: input.Name, CIDR: input.CIDR})
		return err
	case "disk":
		_, err := a.service.CreateDisk(scope.ProjectID, domain.CreateDiskRequest{Name: input.Name, Region: input.Region, Type: input.DiskType, SizeGB: input.SizeGB})
		return err
	case "attachment":
		parent, err := a.service.GetDisk(input.ParentID)
		if err != nil || parent.ProjectID != scope.ProjectID {
			return domain.NotFoundError("disk", input.ParentID)
		}
		_, err = a.service.CreateAttachment(parent, domain.CreateDiskAttachmentRequest{InstanceID: input.InstanceID, Device: input.Device})
		return err
	case "policy":
		actions := strings.Split(input.Actions, ",")
		for i := range actions {
			actions[i] = strings.TrimSpace(actions[i])
		}
		_, err := a.service.CreatePolicy(scope.OrgID, "", domain.CreatePolicyRequest{Name: input.Name, Description: input.Description, Effect: input.Effect, Actions: actions})
		return err
	case "binding":
		parent, err := a.service.GetPolicy(input.ParentID)
		if err != nil || parent.OrgID != scope.OrgID {
			return domain.NotFoundError("policy", input.ParentID)
		}
		_, err = a.service.CreateBinding(parent.ID, domain.CreatePolicyBindingRequest{PrincipalType: input.PrincipalType, PrincipalID: input.PrincipalID, TargetType: input.TargetType, TargetID: input.TargetID})
		return err
	case "load-balancer":
		_, err := a.service.CreateLoadBalancer(scope.ProjectID, domain.CreateLoadBalancerRequest{Name: input.Name, SubnetID: input.SubnetID, Protocol: input.Protocol, Port: input.Port, Algorithm: input.Algorithm, HealthCheckPath: input.HealthCheckPath})
		return err
	case "backend":
		parent, err := a.service.GetLoadBalancer(input.ParentID)
		if err != nil || parent.ProjectID != scope.ProjectID {
			return domain.NotFoundError("load balancer", input.ParentID)
		}
		_, err = a.service.CreateBackend(parent, domain.CreateLoadBalancerBackendRequest{InstanceID: input.InstanceID, Port: input.Port, Weight: input.Weight, Enabled: &input.Enabled})
		return err
	default:
		return domain.InvalidInputError("unsupported graph resource", nil)
	}
}

func (a *serviceGraphConsole) Delete(scope GraphScope, kind, id string) error {
	if err := a.validateScope(scope); err != nil {
		return err
	}
	switch kind {
	case "network":
		value, err := a.service.GetNetwork(id)
		if err != nil || value.ProjectID != scope.ProjectID {
			return domain.NotFoundError(kind, id)
		}
		return a.service.DeleteNetwork(id)
	case "subnet":
		value, err := a.service.GetSubnet(id)
		if err != nil || value.ProjectID != scope.ProjectID {
			return domain.NotFoundError(kind, id)
		}
		return a.service.DeleteSubnet(id)
	case "disk":
		value, err := a.service.GetDisk(id)
		if err != nil || value.ProjectID != scope.ProjectID {
			return domain.NotFoundError(kind, id)
		}
		return a.service.DeleteDisk(id)
	case "attachment":
		value, err := a.service.GetAttachment(id)
		if err != nil || value.ProjectID != scope.ProjectID {
			return domain.NotFoundError(kind, id)
		}
		return a.service.DeleteAttachment(id)
	case "policy":
		value, err := a.service.GetPolicy(id)
		if err != nil || value.OrgID != scope.OrgID {
			return domain.NotFoundError(kind, id)
		}
		return a.service.DeletePolicy(id)
	case "binding":
		value, err := a.service.GetBinding(id)
		if err != nil || value.OrgID != scope.OrgID {
			return domain.NotFoundError(kind, id)
		}
		return a.service.DeleteBinding(id)
	case "load-balancer":
		value, err := a.service.GetLoadBalancer(id)
		if err != nil || value.ProjectID != scope.ProjectID {
			return domain.NotFoundError(kind, id)
		}
		return a.service.DeleteLoadBalancer(id)
	case "backend":
		value, err := a.service.GetBackend(id)
		if err != nil {
			return domain.NotFoundError(kind, id)
		}
		parent, err := a.service.GetLoadBalancer(value.LoadBalancerID)
		if err != nil || parent.ProjectID != scope.ProjectID {
			return domain.NotFoundError(kind, id)
		}
		return a.service.DeleteBackend(id)
	default:
		return domain.InvalidInputError("unsupported graph resource", nil)
	}
}

type GraphSnapshot struct {
	Networks      []GraphNetwork
	Disks         []GraphDisk
	Policies      []GraphPolicy
	LoadBalancers []GraphLoadBalancer
	Instances     []GraphInstance
}

type GraphNetwork struct {
	ID, Name, Region string
	Subnets          []GraphSubnet
}

type GraphSubnet struct {
	ID, Name, CIDR         string
	InstanceCount, LBCount int
}

type GraphAttachment struct {
	ID, InstanceID, InstanceName, Device string
}

type GraphDisk struct {
	ID, Name, Region, Type string
	SizeGB                 int
	Attachment             *GraphAttachment
}

type GraphPolicy struct {
	ID, Name, Description, Effect string
	Actions                       []string
	Bindings                      []GraphBinding
}

type GraphBinding struct {
	ID, PrincipalType, PrincipalID   string
	TargetType, TargetID, TargetName string
}

type GraphLoadBalancer struct {
	ID, Name, SubnetID, SubnetName, Region string
	Protocol, Algorithm, HealthCheckPath   string
	Port                                   int
	Status                                 string
	Backends                               []GraphBackend
}

type GraphBackend struct {
	ID, InstanceID, InstanceName string
	Port, Weight                 int
	Enabled, Healthy             bool
}

type GraphResourceLink struct {
	ID, Name string
}

type GraphLBMembership struct {
	ID, Name, BackendID string
	Port                int
	Healthy             bool
}

type GraphInstance struct {
	ID, Name, Region, Status string
	SubnetID, SubnetName     string
	Disks                    []GraphResourceLink
	LoadBalancers            []GraphLBMembership
}

type GraphCreateInput struct {
	Kind, Name, Region, ParentID, CIDR, DiskType, Device, InstanceID string
	Description, Effect, Actions, PrincipalType, PrincipalID         string
	TargetType, TargetID, SubnetID, Protocol, Algorithm              string
	HealthCheckPath                                                  string
	SizeGB, Port, Weight                                             int
	Enabled                                                          bool
}

func graphScope(org *domain.Organization, project *domain.Project) GraphScope {
	return GraphScope{OrgID: org.ID, ProjectID: project.ID}
}

func (h *Handler) renderCloud(w http.ResponseWriter, r *http.Request, org *domain.Organization, project *domain.Project) {
	ctx, err := h.getPageContext(w, r, org, project, "cloud")
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	graph := GraphSnapshot{}
	state := "empty"
	message := ""
	if ctx.Project != nil {
		graph, err = h.graph.Snapshot(graphScope(org, ctx.Project))
		if err != nil {
			state = "error"
			message = err.Error()
		} else {
			state = "ready"
		}
	}
	h.renderCloudState(w, r, ctx, graph, state, message)
}

func (h *Handler) renderCloudState(w http.ResponseWriter, r *http.Request, ctx *PageContext, graph GraphSnapshot, state, message string) {
	w.Header().Set("Content-Type", "text/html")
	data := map[string]interface{}{
		"CSS": template.CSS(static.CSS), "Context": ctx, "Graph": graph,
		"State": state, "Message": message,
	}
	tmpl := template.Must(template.New("cloud").Parse(baseTemplate + cloudGraphTemplate))
	if isHTMXRequest(r) {
		_ = tmpl.ExecuteTemplate(w, "content", data)
		return
	}
	_ = tmpl.Execute(w, data)
}

func (h *Handler) ListCurrentCloud(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}
	h.renderCloud(w, r, org, nil)
}

func (h *Handler) ListCloud(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}
	h.setCurrentProjectCookie(w, project.Slug)
	h.renderCloud(w, r, org, project)
}

func (h *Handler) NewGraphResourceForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	graph, err := h.graph.Snapshot(graphScope(org, project))
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	kind := r.URL.Query().Get("kind")
	allowed := map[string]bool{
		"network": true, "subnet": true, "disk": true, "attachment": true,
		"policy": true, "binding": true, "load-balancer": true, "backend": true,
	}
	if !allowed[kind] {
		h.renderFormError(w, "Unsupported graph resource")
		return
	}
	sort.Slice(graph.Networks, func(i, j int) bool { return graph.Networks[i].Name < graph.Networks[j].Name })
	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("graph-form").Parse(cloudGraphFormTemplate))
	_ = tmpl.Execute(w, map[string]interface{}{"Project": project, "Graph": graph, "Kind": kind})
}

func (h *Handler) CreateGraphResource(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}
	input := GraphCreateInput{
		Kind: r.FormValue("kind"), Name: r.FormValue("name"), Region: r.FormValue("region"),
		ParentID: r.FormValue("parent_id"), CIDR: r.FormValue("cidr"), DiskType: r.FormValue("disk_type"),
		Device: r.FormValue("device"), InstanceID: r.FormValue("instance_id"), Description: r.FormValue("description"),
		Effect: r.FormValue("effect"), Actions: r.FormValue("actions"), PrincipalType: r.FormValue("principal_type"),
		PrincipalID: r.FormValue("principal_id"), TargetType: r.FormValue("target_type"), TargetID: r.FormValue("target_id"),
		SubnetID: r.FormValue("subnet_id"), Protocol: r.FormValue("protocol"), Algorithm: r.FormValue("algorithm"),
		HealthCheckPath: r.FormValue("health_check_path"), Enabled: r.FormValue("enabled") != "false",
	}
	input.SizeGB, _ = strconv.Atoi(r.FormValue("size_gb"))
	input.Port, _ = strconv.Atoi(r.FormValue("port"))
	input.Weight, _ = strconv.Atoi(r.FormValue("weight"))
	if target := strings.SplitN(r.FormValue("target_ref"), ":", 2); len(target) == 2 {
		input.TargetType, input.TargetID = target[0], target[1]
	}
	if err := h.graph.Create(graphScope(org, project), input); err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	h.renderCloud(w, r, org, project)
}

func (h *Handler) DeleteGraphResource(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	if err := h.graph.Delete(graphScope(org, project), mux.Vars(r)["kind"], mux.Vars(r)["id"]); err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	h.renderCloud(w, r, org, project)
}
