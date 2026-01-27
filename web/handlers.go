package web

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/service"
	"github.com/hypertf/nahcloud/web/static"
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
	Org      *domain.Organization
	Project  *domain.Project
	Orgs     []*domain.Organization
	Projects []*domain.Project
}

// resolveOrg gets the organization from the URL
func (h *Handler) resolveOrg(r *http.Request) (*domain.Organization, error) {
	vars := mux.Vars(r)
	orgSlug := vars["org"]
	if orgSlug == "" {
		return nil, domain.InvalidInputError("organization slug is required", nil)
	}
	return h.service.GetOrganizationBySlug(orgSlug)
}

// resolveProject gets the project from the URL
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

// getPageContext builds the common page context
func (h *Handler) getPageContext(org *domain.Organization, project *domain.Project) (*PageContext, error) {
	orgs, err := h.service.ListOrganizations(domain.OrganizationListOptions{})
	if err != nil {
		return nil, err
	}

	var projects []*domain.Project
	if org != nil {
		projects, err = h.service.ListProjects(domain.ProjectListOptions{OrgID: org.ID})
		if err != nil {
			return nil, err
		}
	}

	return &PageContext{
		Org:      org,
		Project:  project,
		Orgs:     orgs,
		Projects: projects,
	}, nil
}

// ServeLogo serves the static logo
func (h *Handler) ServeLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Write(static.Logo)
}

// Dashboard shows the main dashboard (redirects to default org/project)
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.service.ListOrganizations(domain.OrganizationListOptions{})
	if err != nil || len(orgs) == 0 {
		h.renderError(w, "No organizations found", http.StatusInternalServerError)
		return
	}

	org := orgs[0]

	projects, err := h.service.ListProjects(domain.ProjectListOptions{OrgID: org.ID})
	if err != nil || len(projects) == 0 {
		http.Redirect(w, r, fmt.Sprintf("/org/%s/projects", org.Slug), http.StatusFound)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/org/%s/projects/%s/instances", org.Slug, projects[0].Slug), http.StatusFound)
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

	var defaultProject *domain.Project
	if len(projects) > 0 {
		defaultProject = projects[0]
	}

	ctx, err := h.getPageContext(org, defaultProject)
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
	tmpl.Execute(w, map[string]interface{}{"Org": org})
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

	_, err = h.service.CreateProject(org.ID, req)
	if err != nil {
		h.renderFormError(w, err.Error())
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

	w.WriteHeader(http.StatusOK)
}

// Instance Handlers

// ListInstances handles GET /web/org/{org}/projects/{project}/instances
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx, err := h.getPageContext(org, project)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	instances, err := h.service.ListInstances(domain.InstanceListOptions{ProjectID: project.ID})
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("instances").Parse(baseTemplate + instancesTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":       template.CSS(static.CSS),
		"Context":   ctx,
		"Instances": instances,
	})
}

// NewInstanceForm handles GET /web/org/{org}/projects/{project}/instances/new
func (h *Handler) NewInstanceForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

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
	_, _, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

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

	ctx, err := h.getPageContext(org, nil)
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

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("metadata").Parse(baseTemplate + metadataTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":      template.CSS(static.CSS),
		"Context":  ctx,
		"Metadata": metadata,
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

// ListStorage handles GET /web/org/{org}/projects/{project}/storage
func (h *Handler) ListStorage(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx, err := h.getPageContext(org, project)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buckets, err := h.service.ListBuckets(domain.BucketListOptions{ProjectID: project.ID})
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("storage").Parse(baseTemplate + storageTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"CSS":     template.CSS(static.CSS),
		"Context": ctx,
		"Buckets": buckets,
	})
}

// NewBucketForm handles GET /web/org/{org}/projects/{project}/storage/buckets/new
func (h *Handler) NewBucketForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderFormError(w, err.Error())
		return
	}

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

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.renderError(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx, err := h.getPageContext(org, project)
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

