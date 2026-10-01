package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/web/static"
)

// GraphConsole is the deliberately narrow integration boundary for the proposed
// graph backend. Backend-owned domain/service types can implement this contract
// later without leaking persistence details into the console.
type GraphConsole interface {
	Snapshot(projectID string) (GraphSnapshot, error)
	Create(projectID string, input GraphCreateInput) error
	Delete(projectID, kind, id string) error
	SetFault(projectID, fault string, enabled bool) error
}

type GraphSnapshot struct {
	Networks      []GraphNetwork
	Disks         []GraphDisk
	Policies      []GraphPolicy
	LoadBalancers []GraphLoadBalancer
	Instances     []GraphInstance
	Faults        []GraphFault
}

type GraphNetwork struct {
	ID, Name, CIDR, Region string
	Subnets                []GraphSubnet
}
type GraphSubnet struct{ ID, Name, CIDR, Zone string }
type GraphDisk struct{ ID, Name, Size, Region, InstanceID, InstanceName, MountPath string }
type GraphPolicy struct {
	ID, Name, Effect string
	Bindings         []GraphBinding
}
type GraphBinding struct{ ID, Principal, Role, TargetKind, TargetID, TargetName string }
type GraphLoadBalancer struct {
	ID, Name, Address, Protocol string
	Backends                    []GraphBackend
}
type GraphBackend struct{ ID, InstanceID, InstanceName, Port, Health string }
type GraphInstance struct{ ID, Name, Region string }
type GraphFault struct {
	Kind, Label, Description string
	Enabled                  bool
}

type GraphCreateInput struct {
	Kind, Name, CIDR, Region, ParentID, Zone, Size, InstanceID, MountPath  string
	Effect, Principal, Role, TargetKind, TargetID, Address, Protocol, Port string
}

var errGraphResourceNotFound = errors.New("graph resource not found in project")

type previewGraphConsole struct {
	mu       sync.Mutex
	projects map[string]GraphSnapshot
}

func newPreviewGraphConsole() *previewGraphConsole {
	return &previewGraphConsole{projects: make(map[string]GraphSnapshot)}
}

func (p *previewGraphConsole) Snapshot(projectID string) (GraphSnapshot, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	graph := p.project(projectID)
	return graph, nil
}

func (p *previewGraphConsole) project(projectID string) GraphSnapshot {
	if graph, ok := p.projects[projectID]; ok {
		return graph
	}
	// Include the complete project ID so preview references cannot collide across
	// tenants that happen to share a common ID prefix.
	short := projectID
	graph := GraphSnapshot{
		Instances:     []GraphInstance{{ID: "vm-web-" + short, Name: "web-01", Region: "us-east-1"}, {ID: "vm-api-" + short, Name: "api-01", Region: "us-east-1"}},
		Networks:      []GraphNetwork{{ID: "net-" + short, Name: "production", CIDR: "10.24.0.0/16", Region: "us-east-1", Subnets: []GraphSubnet{{ID: "subnet-public-" + short, Name: "public-a", CIDR: "10.24.1.0/24", Zone: "us-east-1a"}, {ID: "subnet-private-" + short, Name: "private-a", CIDR: "10.24.10.0/24", Zone: "us-east-1a"}}}},
		Disks:         []GraphDisk{{ID: "disk-" + short, Name: "api-data", Size: "100 GiB", Region: "us-east-1", InstanceID: "vm-api-" + short, InstanceName: "api-01", MountPath: "/var/lib/app"}},
		Policies:      []GraphPolicy{{ID: "policy-" + short, Name: "service-operators", Effect: "allow", Bindings: []GraphBinding{{ID: "binding-" + short, Principal: "team:platform", Role: "operator", TargetKind: "network", TargetID: "net-" + short, TargetName: "production"}}}},
		LoadBalancers: []GraphLoadBalancer{{ID: "lb-" + short, Name: "public-api", Address: "203.0.113.42", Protocol: "HTTPS :443", Backends: []GraphBackend{{ID: "backend-web-" + short, InstanceID: "vm-web-" + short, InstanceName: "web-01", Port: "8080", Health: "healthy"}, {ID: "backend-api-" + short, InstanceID: "vm-api-" + short, InstanceName: "api-01", Port: "8080", Health: "healthy"}}}},
		Faults:        []GraphFault{{Kind: "latency", Label: "Network latency", Description: "Add 250 ms latency inside this tenant."}, {Kind: "packet-loss", Label: "Packet loss", Description: "Drop 10% of tenant network traffic."}, {Kind: "backend-outage", Label: "Backend outage", Description: "Mark load-balancer backends unhealthy."}},
	}
	p.projects[projectID] = graph
	return graph
}

func (p *previewGraphConsole) Create(projectID string, in GraphCreateInput) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	g := p.project(projectID)
	id := in.Kind + "-" + uuid.NewString()[:8]
	switch in.Kind {
	case "network":
		g.Networks = append(g.Networks, GraphNetwork{ID: id, Name: in.Name, CIDR: in.CIDR, Region: in.Region})
	case "subnet":
		for i := range g.Networks {
			if g.Networks[i].ID == in.ParentID {
				g.Networks[i].Subnets = append(g.Networks[i].Subnets, GraphSubnet{ID: id, Name: in.Name, CIDR: in.CIDR, Zone: in.Zone})
				p.projects[projectID] = g
				return nil
			}
		}
		return errGraphResourceNotFound
	case "disk":
		instance, ok := graphInstance(g, in.InstanceID)
		if in.InstanceID != "" && !ok {
			return errGraphResourceNotFound
		}
		g.Disks = append(g.Disks, GraphDisk{ID: id, Name: in.Name, Size: in.Size, Region: in.Region, InstanceID: in.InstanceID, InstanceName: instance.Name, MountPath: in.MountPath})
	case "policy":
		g.Policies = append(g.Policies, GraphPolicy{ID: id, Name: in.Name, Effect: in.Effect})
	case "binding":
		targetName, ok := graphTarget(g, in.TargetKind, in.TargetID)
		if !ok {
			return errGraphResourceNotFound
		}
		for i := range g.Policies {
			if g.Policies[i].ID == in.ParentID {
				g.Policies[i].Bindings = append(g.Policies[i].Bindings, GraphBinding{ID: id, Principal: in.Principal, Role: in.Role, TargetKind: in.TargetKind, TargetID: in.TargetID, TargetName: targetName})
				p.projects[projectID] = g
				return nil
			}
		}
		return errGraphResourceNotFound
	case "load-balancer":
		g.LoadBalancers = append(g.LoadBalancers, GraphLoadBalancer{ID: id, Name: in.Name, Address: in.Address, Protocol: in.Protocol})
	case "backend":
		instance, ok := graphInstance(g, in.InstanceID)
		if !ok {
			return errGraphResourceNotFound
		}
		for i := range g.LoadBalancers {
			if g.LoadBalancers[i].ID == in.ParentID {
				g.LoadBalancers[i].Backends = append(g.LoadBalancers[i].Backends, GraphBackend{ID: id, InstanceID: instance.ID, InstanceName: instance.Name, Port: in.Port, Health: "pending"})
				p.projects[projectID] = g
				return nil
			}
		}
		return errGraphResourceNotFound
	default:
		return fmt.Errorf("unsupported graph resource %q", in.Kind)
	}
	p.projects[projectID] = g
	return nil
}

func graphInstance(g GraphSnapshot, id string) (GraphInstance, bool) {
	for _, instance := range g.Instances {
		if instance.ID == id {
			return instance, true
		}
	}
	return GraphInstance{}, false
}

func graphTarget(g GraphSnapshot, kind, id string) (string, bool) {
	switch kind {
	case "network":
		for _, item := range g.Networks {
			if item.ID == id {
				return item.Name, true
			}
		}
	case "disk":
		for _, item := range g.Disks {
			if item.ID == id {
				return item.Name, true
			}
		}
	case "load-balancer":
		for _, item := range g.LoadBalancers {
			if item.ID == id {
				return item.Name, true
			}
		}
	case "instance":
		if item, ok := graphInstance(g, id); ok {
			return item.Name, true
		}
	}
	return "", false
}

func (p *previewGraphConsole) Delete(projectID, kind, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	g := p.project(projectID)
	removed := false
	switch kind {
	case "network":
		g.Networks, removed = deleteNetwork(g.Networks, id)
	case "subnet":
		for i := range g.Networks {
			var hit bool
			g.Networks[i].Subnets, hit = deleteSubnet(g.Networks[i].Subnets, id)
			removed = removed || hit
		}
	case "disk":
		g.Disks, removed = deleteDisk(g.Disks, id)
	case "policy":
		g.Policies, removed = deletePolicy(g.Policies, id)
	case "binding":
		for i := range g.Policies {
			var hit bool
			g.Policies[i].Bindings, hit = deleteBinding(g.Policies[i].Bindings, id)
			removed = removed || hit
		}
	case "load-balancer":
		g.LoadBalancers, removed = deleteLoadBalancer(g.LoadBalancers, id)
	case "backend":
		for i := range g.LoadBalancers {
			var hit bool
			g.LoadBalancers[i].Backends, hit = deleteBackend(g.LoadBalancers[i].Backends, id)
			removed = removed || hit
		}
	}
	if !removed {
		return errGraphResourceNotFound
	}
	p.projects[projectID] = g
	return nil
}

func deleteNetwork(items []GraphNetwork, id string) ([]GraphNetwork, bool) {
	for i, v := range items {
		if v.ID == id {
			return append(items[:i], items[i+1:]...), true
		}
	}
	return items, false
}
func deleteSubnet(items []GraphSubnet, id string) ([]GraphSubnet, bool) {
	for i, v := range items {
		if v.ID == id {
			return append(items[:i], items[i+1:]...), true
		}
	}
	return items, false
}
func deleteDisk(items []GraphDisk, id string) ([]GraphDisk, bool) {
	for i, v := range items {
		if v.ID == id {
			return append(items[:i], items[i+1:]...), true
		}
	}
	return items, false
}
func deletePolicy(items []GraphPolicy, id string) ([]GraphPolicy, bool) {
	for i, v := range items {
		if v.ID == id {
			return append(items[:i], items[i+1:]...), true
		}
	}
	return items, false
}
func deleteBinding(items []GraphBinding, id string) ([]GraphBinding, bool) {
	for i, v := range items {
		if v.ID == id {
			return append(items[:i], items[i+1:]...), true
		}
	}
	return items, false
}
func deleteLoadBalancer(items []GraphLoadBalancer, id string) ([]GraphLoadBalancer, bool) {
	for i, v := range items {
		if v.ID == id {
			return append(items[:i], items[i+1:]...), true
		}
	}
	return items, false
}
func deleteBackend(items []GraphBackend, id string) ([]GraphBackend, bool) {
	for i, v := range items {
		if v.ID == id {
			return append(items[:i], items[i+1:]...), true
		}
	}
	return items, false
}

func (p *previewGraphConsole) SetFault(projectID, fault string, enabled bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	g := p.project(projectID)
	for i := range g.Faults {
		if g.Faults[i].Kind == fault {
			g.Faults[i].Enabled = enabled
			p.projects[projectID] = g
			return nil
		}
	}
	return errGraphResourceNotFound
}

func (h *Handler) renderCloud(w http.ResponseWriter, r *http.Request, org *domain.Organization, project *domain.Project) {
	ctx, err := h.getPageContext(w, r, org, project, "cloud")
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	state := r.URL.Query().Get("state")
	if state == "error" {
		h.renderCloudState(w, r, ctx, GraphSnapshot{}, state, "Graph service could not be reached. Existing resources were not changed.")
		return
	}
	graph := GraphSnapshot{}
	if ctx.Project != nil && state != "empty" && state != "loading" {
		graph, err = h.graph.Snapshot(ctx.Project.ID)
		if err != nil {
			h.renderCloudState(w, r, ctx, graph, "error", err.Error())
			return
		}
	}
	h.renderCloudState(w, r, ctx, graph, state, "")
}

func (h *Handler) renderCloudState(w http.ResponseWriter, r *http.Request, ctx *PageContext, graph GraphSnapshot, state, message string) {
	w.Header().Set("Content-Type", "text/html")
	data := map[string]interface{}{"CSS": template.CSS(static.CSS), "Context": ctx, "Graph": graph, "State": state, "Message": message}
	tmpl := template.Must(template.New("cloud").Funcs(template.FuncMap{"lower": strings.ToLower}).Parse(baseTemplate + cloudGraphTemplate))
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
	_, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	graph, err := h.graph.Snapshot(project.ID)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	kind := r.URL.Query().Get("kind")
	allowed := map[string]bool{"network": true, "subnet": true, "disk": true, "policy": true, "binding": true, "load-balancer": true, "backend": true}
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
	in := GraphCreateInput{Kind: r.FormValue("kind"), Name: r.FormValue("name"), CIDR: r.FormValue("cidr"), Region: r.FormValue("region"), ParentID: r.FormValue("parent_id"), Zone: r.FormValue("zone"), Size: r.FormValue("size"), InstanceID: r.FormValue("instance_id"), MountPath: r.FormValue("mount_path"), Effect: r.FormValue("effect"), Principal: r.FormValue("principal"), Role: r.FormValue("role"), TargetKind: r.FormValue("target_kind"), TargetID: r.FormValue("target_id"), Address: r.FormValue("address"), Protocol: r.FormValue("protocol"), Port: r.FormValue("port")}
	if target := strings.SplitN(r.FormValue("target_ref"), ":", 2); len(target) == 2 {
		in.TargetKind, in.TargetID = target[0], target[1]
	}
	if err := h.graph.Create(project.ID, in); err != nil {
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
	if err := h.graph.Delete(project.ID, mux.Vars(r)["kind"], mux.Vars(r)["id"]); err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	h.renderCloud(w, r, org, project)
}

func (h *Handler) SetGraphFault(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}
	if err := h.graph.SetFault(project.ID, mux.Vars(r)["fault"], r.FormValue("enabled") == "true"); err != nil {
		h.renderFormError(w, err.Error())
		return
	}
	h.renderCloud(w, r, org, project)
}
