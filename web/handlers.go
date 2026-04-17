package web

import (
	"encoding/base64"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/pkg/auth"
	"github.com/hypertf/nahcloud/service"
	"github.com/hypertf/nahcloud/web/static"
)

const (
	sessionCookieName    = "nah_session"
	sessionCookieMaxAge  = 30 * 24 * 60 * 60
	currentProjectCookie = "nah_project"
)

// Handler handles web console requests
type Handler struct {
	service *service.Service
}

// NewHandler creates a new web handler
func NewHandler(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

// PageContext contains common data for all pages
type PageContext struct {
	Org                         *domain.Organization
	Project                     *domain.Project
	Projects                    []*domain.Project
	Section                     string
	CurrentProjectInstanceCount int
	CurrentProjectBucketCount   int
	MetadataCount               int
}

// resolveOrg gets the organization from context (set by middleware)
func (h *Handler) resolveOrg(r *http.Request) (*domain.Organization, error) {
	org := auth.OrgFromContext(r.Context())
	if org == nil {
		return nil, domain.UnauthorizedError("no organization in context")
	}
	return org, nil
}

// resolveProject gets the project from the URL (project slug still comes from URL)
func (h *Handler) resolveProject(r *http.Request) (*domain.Organization, *domain.Project, error) {
	org, err := h.resolveOrg(r)
	if err != nil {
		return nil, nil, err
	}

	vars := mux.Vars(r)
	projectSlug := vars["project"]
	if projectSlug == "" {
		return org, nil, domain.InvalidInputError("project slug is required", nil)
	}

	project, err := h.service.GetProjectBySlug(org.ID, projectSlug)
	if err != nil {
		return org, nil, err
	}

	return org, project, nil
}

func (h *Handler) resolveCurrentProject(w http.ResponseWriter, r *http.Request, org *domain.Organization) (*domain.Project, error) {
	if org == nil {
		return nil, nil
	}

	if cookie, err := r.Cookie(currentProjectCookie); err == nil && cookie.Value != "" {
		project, err := h.service.GetProjectBySlug(org.ID, cookie.Value)
		if err == nil {
			return project, nil
		}
	}

	projects, err := h.service.ListProjects(domain.ProjectListOptions{OrgID: org.ID})
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		h.clearCurrentProjectCookie(w)
		return nil, nil
	}

	h.setCurrentProjectCookie(w, projects[0].Slug)
	return projects[0], nil
}

// getPageContext builds the common page context.
func (h *Handler) getPageContext(w http.ResponseWriter, r *http.Request, org *domain.Organization, project *domain.Project, section string) (*PageContext, error) {
	var (
		projects []*domain.Project
		err      error
	)
	if org != nil {
		projects, err = h.service.ListProjects(domain.ProjectListOptions{OrgID: org.ID})
		if err != nil {
			return nil, err
		}
	}

	if project == nil {
		project, err = h.resolveCurrentProject(w, r, org)
		if err != nil {
			return nil, err
		}
	}

	instanceCount := 0
	bucketCount := 0
	if project != nil {
		instances, err := h.service.ListInstances(domain.InstanceListOptions{ProjectID: project.ID})
		if err != nil {
			return nil, err
		}
		instanceCount = len(instances)

		buckets, err := h.service.ListBuckets(domain.BucketListOptions{ProjectID: project.ID})
		if err != nil {
			return nil, err
		}
		bucketCount = len(buckets)
	}

	metadataCount := 0
	if org != nil {
		metadata, err := h.service.ListMetadata(domain.MetadataListOptions{OrgID: org.ID})
		if err != nil {
			return nil, err
		}
		metadataCount = len(visibleMetadata(metadata))
	}

	return &PageContext{
		Org:                         org,
		Project:                     project,
		Projects:                    projects,
		Section:                     section,
		CurrentProjectInstanceCount: instanceCount,
		CurrentProjectBucketCount:   bucketCount,
		MetadataCount:               metadataCount,
	}, nil
}

func (h *Handler) organizationPermalink(r *http.Request, slug string) string {
	scheme := "http"
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	} else if r.TLS != nil {
		scheme = "https"
	}

	return scheme + "://" + r.Host + "/o/" + slug
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   sessionCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) setCurrentProjectCookie(w http.ResponseWriter, slug string) {
	http.SetCookie(w, &http.Cookie{
		Name:     currentProjectCookie,
		Value:    slug,
		Path:     "/",
		MaxAge:   sessionCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearCurrentProjectCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     currentProjectCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) currentSectionRedirect(section string) string {
	switch section {
	case "overview":
		return "/instances"
	case "instances":
		return "/instances"
	case "storage":
		return "/storage"
	case "metadata":
		return "/metadata"
	case "projects":
		return "/projects"
	case "settings":
		return "/settings"
	default:
		return "/instances"
	}
}

func visibleMetadata(metadata []*domain.Metadata) []*domain.Metadata {
	filtered := make([]*domain.Metadata, 0, len(metadata))
	for _, item := range metadata {
		if strings.HasPrefix(item.Path, ".nahcloud/") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// ServeLogo serves the static logo
func (h *Handler) ServeLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Write(static.Logo)
}

// OpenPermalink switches the browser into the requested organization and creates a session.
func (h *Handler) OpenPermalink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgSlug := vars["org"]
	if orgSlug == "" {
		h.renderError(w, "Organization link is invalid", http.StatusBadRequest)
		return
	}

	org, err := h.service.GetOrganizationBySlug(orgSlug)
	if err != nil {
		status := http.StatusInternalServerError
		if domain.IsNotFound(err) {
			status = http.StatusNotFound
		}
		h.renderError(w, err.Error(), status)
		return
	}

	session, err := h.service.CreateSession(org.ID)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, session.Token)
	http.Redirect(w, r, "/", http.StatusFound)
}

// Settings shows share and reset controls for the current organization.
func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx, err := h.getPageContext(w, r, org, nil, "settings")
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("settings").Parse(baseTemplate + settingsTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":       template.CSS(static.CSS),
		"Context":   ctx,
		"Permalink": h.organizationPermalink(r, org.Slug),
		"ResetDone": r.URL.Query().Get("reset") == "1",
	})
}

// ResetOrganization clears the current organization back to its initial blank state.
func (h *Handler) ResetOrganization(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := h.service.ResetOrganization(org.ID); err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.clearCurrentProjectCookie(w)
	http.Redirect(w, r, "/settings?reset=1", http.StatusSeeOther)
}

// SelectProject updates the current project cookie and redirects to the requested section.
func (h *Handler) SelectProject(w http.ResponseWriter, r *http.Request) {
	_, project, err := h.resolveProject(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)
	http.Redirect(w, r, h.currentSectionRedirect(r.URL.Query().Get("next")), http.StatusFound)
}

// renderError renders a full page error
func (h *Handler) renderError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	tmpl := template.Must(template.New("error").Parse(errorTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":     template.CSS(static.CSS),
		"Message": message,
	})
}

// renderFormError renders an error banner in forms (targets #form-error)
func (h *Handler) renderFormError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Retarget", "#form-error")
	w.Header().Set("HX-Reswap", "innerHTML")
	w.WriteHeader(http.StatusBadRequest)
	html := `<div class="flex items-center gap-2 p-3 bg-red-50 border border-red-200 rounded-lg">
<svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
</svg>
<span class="text-sm text-red-700">` + template.HTMLEscapeString(message) + `</span>
</div>`
	w.Write([]byte(html))
}

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Project Handlers

// ListProjects handles GET /web/org/{org}/projects
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	projects, err := h.service.ListProjects(domain.ProjectListOptions{OrgID: org.ID})
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, err := h.getPageContext(w, r, org, nil, "projects")
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("projects").Parse(baseTemplate + projectsTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":      template.CSS(static.CSS),
		"Context":  ctx,
		"Projects": projects,
	})
}

// NewProjectForm handles GET /web/org/{org}/projects/new
func (h *Handler) NewProjectForm(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("new-project").Parse(newProjectFormTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"Org":        org,
		"RedirectTo": r.URL.Query().Get("redirect_to"),
	})
}

// CreateProject handles POST /web/org/{org}/projects
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	req := domain.CreateProjectRequest{
		Slug: r.FormValue("slug"),
		Name: r.FormValue("name"),
	}

	createdProject, err := h.service.CreateProject(org.ID, req)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	if createdProject != nil {
		h.setCurrentProjectCookie(w, createdProject.Slug)
	}

	if isHTMXRequest(r) {
		redirectTo := r.FormValue("redirect_to")
		if redirectTo == "" {
			redirectTo = "/instances"
		}
		w.Header().Set("HX-Redirect", redirectTo)
		w.WriteHeader(http.StatusOK)
		return
	}

	h.ListProjects(w, r)
}

// EditProjectForm handles GET /web/org/{org}/projects/{project}/edit
func (h *Handler) EditProjectForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)
	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("edit-project").Parse(editProjectFormTemplate))
	tmpl.Execute(w, map[string]interface{}{"Org": org, "Project": project})
}

// UpdateProject handles PUT /web/org/{org}/projects/{project}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	_, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	name := r.FormValue("name")
	req := domain.UpdateProjectRequest{Name: &name}

	_, err = h.service.UpdateProject(project.ID, req)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.ListProjects(w, r)
}

// DeleteProject handles DELETE /web/org/{org}/projects/{project}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	_, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	if err := h.service.DeleteProject(project.ID); err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	if cookie, err := r.Cookie(currentProjectCookie); err == nil && cookie.Value == project.Slug {
		h.clearCurrentProjectCookie(w)
	}
	w.WriteHeader(http.StatusOK)
}

// Instance Handlers

func (h *Handler) renderInstancesPage(w http.ResponseWriter, r *http.Request, org *domain.Organization, project *domain.Project) {
	ctx, err := h.getPageContext(w, r, org, project, "instances")
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var instances []*domain.Instance
	if ctx.Project != nil {
		instances, err = h.service.ListInstances(domain.InstanceListOptions{ProjectID: ctx.Project.ID})
		if err != nil {
			h.renderError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("instances").Parse(baseTemplate + instancesTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":       template.CSS(static.CSS),
		"Context":   ctx,
		"Instances": instances,
	})
}

// ListInstances handles GET /projects/{project}/instances and pins that project as current.
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)
	h.renderInstancesPage(w, r, org, project)
}

// ListCurrentInstances handles GET /instances.
func (h *Handler) ListCurrentInstances(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.renderInstancesPage(w, r, org, nil)
}

// NewInstanceForm handles GET /web/org/{org}/projects/{project}/instances/new
func (h *Handler) NewInstanceForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)
	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("new-instance").Parse(newInstanceFormTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"Org":     org,
		"Project": project,
		"Regions": domain.ValidRegions,
	})
}

// CreateInstance handles POST /web/org/{org}/projects/{project}/instances
func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	_, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	cpu, _ := strconv.Atoi(r.FormValue("cpu"))
	memoryMB, _ := strconv.Atoi(r.FormValue("memory_mb"))

	req := domain.CreateInstanceRequest{
		ProjectID: project.ID,
		Name:      r.FormValue("name"),
		Region:    r.FormValue("region"),
		CPU:       cpu,
		MemoryMB:  memoryMB,
		Image:     r.FormValue("image"),
		Status:    r.FormValue("status"),
	}

	_, err = h.service.CreateInstance(req)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.ListInstances(w, r)
}

// EditInstanceForm handles GET /web/org/{org}/projects/{project}/instances/{id}/edit
func (h *Handler) EditInstanceForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)

	vars := mux.Vars(r)
	id := vars["id"]

	instance, err := h.service.GetInstance(id)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("edit-instance").Parse(editInstanceFormTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"Org":      org,
		"Project":  project,
		"Instance": instance,
		"Regions":  domain.ValidRegions,
	})
}

// UpdateInstance handles PUT /web/org/{org}/projects/{project}/instances/{id}
func (h *Handler) UpdateInstance(w http.ResponseWriter, r *http.Request) {
	_, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)

	vars := mux.Vars(r)
	id := vars["id"]

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	name := r.FormValue("name")
	cpu, _ := strconv.Atoi(r.FormValue("cpu"))
	memoryMB, _ := strconv.Atoi(r.FormValue("memory_mb"))
	status := r.FormValue("status")

	req := domain.UpdateInstanceRequest{
		Name:     &name,
		CPU:      &cpu,
		MemoryMB: &memoryMB,
		Status:   &status,
	}

	_, err = h.service.UpdateInstance(id, req)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.ListInstances(w, r)
}

// DeleteInstance handles DELETE /web/org/{org}/projects/{project}/instances/{id}
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.service.DeleteInstance(id); err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Metadata Handlers

// ListMetadata handles GET /web/org/{org}/metadata
func (h *Handler) ListMetadata(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx, err := h.getPageContext(w, r, org, nil, "metadata")
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prefix := r.URL.Query().Get("prefix")
	metadata, err := h.service.ListMetadata(domain.MetadataListOptions{OrgID: org.ID, Prefix: prefix})
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filtered := metadata[:0]
	for _, item := range metadata {
		if strings.HasPrefix(item.Path, ".nahcloud/") && !strings.HasPrefix(prefix, ".nahcloud/") {
			continue
		}
		filtered = append(filtered, item)
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("metadata").Parse(baseTemplate + metadataTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":      template.CSS(static.CSS),
		"Context":  ctx,
		"Metadata": filtered,
		"Prefix":   prefix,
	})
}

// NewMetadataForm handles GET /web/org/{org}/metadata/new
func (h *Handler) NewMetadataForm(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("new-metadata").Parse(newMetadataFormTemplate))
	tmpl.Execute(w, map[string]interface{}{"Org": org})
}

// CreateMetadata handles POST /web/org/{org}/metadata
func (h *Handler) CreateMetadata(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	req := domain.CreateMetadataRequest{
		OrgID: org.ID,
		Path:  r.FormValue("path"),
		Value: r.FormValue("value"),
	}

	_, err = h.service.CreateMetadata(req)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.ListMetadata(w, r)
}

// EditMetadataForm handles GET /web/org/{org}/metadata/edit
func (h *Handler) EditMetadataForm(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		h.renderFormError(w, "Metadata ID is required")
		return
	}

	metadata, err := h.service.GetMetadata(id)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("edit-metadata").Parse(editMetadataFormTemplate))
	tmpl.Execute(w, map[string]interface{}{"Org": org, "Metadata": metadata})
}

// UpdateMetadata handles PUT /web/org/{org}/metadata/update
func (h *Handler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	_, err := h.resolveOrg(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	id := r.FormValue("id")
	value := r.FormValue("value")

	req := domain.UpdateMetadataRequest{Value: &value}

	_, err = h.service.UpdateMetadata(id, req)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.ListMetadata(w, r)
}

// DeleteMetadata handles DELETE /web/org/{org}/metadata/delete
func (h *Handler) DeleteMetadata(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		h.renderFormError(w, "Metadata ID is required")
		return
	}

	if err := h.service.DeleteMetadata(id); err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Storage Handlers

func (h *Handler) renderStoragePage(w http.ResponseWriter, r *http.Request, org *domain.Organization, project *domain.Project) {
	ctx, err := h.getPageContext(w, r, org, project, "storage")
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var buckets []*domain.Bucket
	if ctx.Project != nil {
		buckets, err = h.service.ListBuckets(domain.BucketListOptions{ProjectID: ctx.Project.ID})
		if err != nil {
			h.renderError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("storage").Parse(baseTemplate + storageTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":     template.CSS(static.CSS),
		"Context": ctx,
		"Buckets": buckets,
	})
}

// ListStorage handles GET /projects/{project}/storage and pins that project as current.
func (h *Handler) ListStorage(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)
	h.renderStoragePage(w, r, org, project)
}

// ListCurrentStorage handles GET /storage.
func (h *Handler) ListCurrentStorage(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.renderStoragePage(w, r, org, nil)
}

// NewBucketForm handles GET /web/org/{org}/projects/{project}/storage/buckets/new
func (h *Handler) NewBucketForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)
	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("new-bucket").Parse(newBucketFormTemplate))
	tmpl.Execute(w, map[string]interface{}{"Org": org, "Project": project})
}

// CreateBucket handles POST /web/org/{org}/projects/{project}/storage/buckets
func (h *Handler) CreateBucket(w http.ResponseWriter, r *http.Request) {
	_, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	req := domain.CreateBucketRequest{
		Name: r.FormValue("name"),
	}

	_, err = h.service.CreateBucket(project.ID, req)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.ListStorage(w, r)
}

// ListBucketObjects handles GET /web/org/{org}/projects/{project}/storage/{bucket}
func (h *Handler) ListBucketObjects(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx, err := h.getPageContext(w, r, org, project, "storage")
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prefix := r.URL.Query().Get("prefix")
	objects, err := h.service.ListObjects(domain.ObjectListOptions{BucketID: bucket.ID, Prefix: prefix})
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("bucket-objects").Parse(baseTemplate + bucketObjectsTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":     template.CSS(static.CSS),
		"Context": ctx,
		"Bucket":  bucket,
		"Objects": objects,
		"Prefix":  prefix,
	})
}

// NewObjectForm handles GET /web/org/{org}/projects/{project}/storage/{bucket}/objects/new
func (h *Handler) NewObjectForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("new-object").Parse(newObjectFormTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"Org":     org,
		"Project": project,
		"Bucket":  bucket,
	})
}

// CreateObject handles POST /web/org/{org}/projects/{project}/storage/{bucket}/objects
func (h *Handler) CreateObject(w http.ResponseWriter, r *http.Request) {
	_, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	content := r.FormValue("content")
	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	req := domain.CreateObjectRequest{
		BucketID: bucket.ID,
		Path:     r.FormValue("path"),
		Content:  encoded,
	}

	_, err = h.service.CreateObject(req)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.ListBucketObjects(w, r)
}

// ViewObject handles GET /web/org/{org}/projects/{project}/storage/{bucket}/objects/{objid}
func (h *Handler) ViewObject(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	h.setCurrentProjectCookie(w, project.Slug)

	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	objID := vars["objid"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	obj, err := h.service.GetObject(objID)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	decoded, _ := base64.StdEncoding.DecodeString(obj.Content)

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("view-object").Parse(viewObjectTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"Org":            org,
		"Project":        project,
		"Bucket":         bucket,
		"Object":         obj,
		"DecodedContent": string(decoded),
		"Size":           len(decoded),
	})
}
