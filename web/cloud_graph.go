package web

import (
	"errors"
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

// ErrGraphBackendUnavailable is returned until the corrected graph service
// contract from PR #4 lands on main. The web console deliberately shows an
// unavailable state instead of inventing runtime resources.
var ErrGraphBackendUnavailable = errors.New("cloud graph backend is not available in this build")

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

func (a *serviceGraphConsole) Snapshot(GraphScope) (GraphSnapshot, error) {
	// Keep the real service dependency explicit. This method will map corrected
	// domain graph types after PR #4 freezes; it must never synthesize resources.
	_ = a.service
	return GraphSnapshot{}, ErrGraphBackendUnavailable
}

func (a *serviceGraphConsole) Create(GraphScope, GraphCreateInput) error {
	return ErrGraphBackendUnavailable
}

func (a *serviceGraphConsole) Delete(GraphScope, string, string) error {
	return ErrGraphBackendUnavailable
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
