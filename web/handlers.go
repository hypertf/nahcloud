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
func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
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

// Dashboard shows the main dashboard (redirects to default org)
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Get default org
	orgs, err := h.service.ListOrganizations(domain.OrganizationListOptions{})
	if err != nil || len(orgs) == 0 {
		h.renderError(w, "No organizations found", http.StatusInternalServerError)
		return
	}

	// Redirect to default org's projects
	http.Redirect(w, r, fmt.Sprintf("/web/org/%s/projects", orgs[0].Slug), http.StatusFound)
}

// renderError renders an error message
func (h *Handler) renderError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	tmpl := template.Must(template.New("error").Parse(errorTemplate))
	tmpl.Execute(w, map[string]interface{}{"Message": message})
}

// renderErrorBanner renders an error banner for HTMX requests
func (h *Handler) renderErrorBanner(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Retarget", "#error-container")
	w.Header().Set("HX-Reswap", "innerHTML")
	tmpl := template.Must(template.New("error-banner").Parse(errorBannerTemplate))
	tmpl.Execute(w, map[string]interface{}{"Message": message})
}

// Project Handlers

// ListProjects handles GET /web/org/{org}/projects
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
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

	projects, err := h.service.ListProjects(domain.ProjectListOptions{OrgID: org.ID})
	if err != nil {
		h.renderError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("projects").Parse(baseTemplate + projectsTemplate))
	tmpl.Execute(w, map[string]interface{}{
		"Context":  ctx,
		"Projects": projects,
	})
}

// NewProjectForm handles GET /web/org/{org}/projects/new
func (h *Handler) NewProjectForm(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
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
		h.renderErrorBanner(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderErrorBanner(w, "Failed to parse form")
		return
	}

	req := domain.CreateProjectRequest{
		Slug: r.FormValue("slug"),
		Name: r.FormValue("name"),
	}

	_, err = h.service.CreateProject(org.ID, req)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/projects", org.Slug))
	w.WriteHeader(http.StatusOK)
}

// EditProjectForm handles GET /web/org/{org}/projects/{project}/edit
func (h *Handler) EditProjectForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("edit-project").Parse(editProjectFormTemplate))
	tmpl.Execute(w, map[string]interface{}{"Org": org, "Project": project})
}

// UpdateProject handles PUT /web/org/{org}/projects/{project}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderErrorBanner(w, "Failed to parse form")
		return
	}

	name := r.FormValue("name")
	req := domain.UpdateProjectRequest{Name: &name}

	_, err = h.service.UpdateProject(project.ID, req)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/projects", org.Slug))
	w.WriteHeader(http.StatusOK)
}

// DeleteProject handles DELETE /web/org/{org}/projects/{project}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	if err := h.service.DeleteProject(project.ID); err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/projects", org.Slug))
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
		"Context":   ctx,
		"Instances": instances,
	})
}

// NewInstanceForm handles GET /web/org/{org}/projects/{project}/instances/new
func (h *Handler) NewInstanceForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
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
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderErrorBanner(w, "Failed to parse form")
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
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/projects/%s/instances", org.Slug, project.Slug))
	w.WriteHeader(http.StatusOK)
}

// EditInstanceForm handles GET /web/org/{org}/projects/{project}/instances/{id}/edit
func (h *Handler) EditInstanceForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	instance, err := h.service.GetInstance(id)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
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
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if err := r.ParseForm(); err != nil {
		h.renderErrorBanner(w, "Failed to parse form")
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
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/projects/%s/instances", org.Slug, project.Slug))
	w.WriteHeader(http.StatusOK)
}

// DeleteInstance handles DELETE /web/org/{org}/projects/{project}/instances/{id}
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.service.DeleteInstance(id); err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/projects/%s/instances", org.Slug, project.Slug))
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
		"Context":  ctx,
		"Metadata": metadata,
		"Prefix":   prefix,
	})
}

// NewMetadataForm handles GET /web/org/{org}/metadata/new
func (h *Handler) NewMetadataForm(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
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
		h.renderErrorBanner(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderErrorBanner(w, "Failed to parse form")
		return
	}

	req := domain.CreateMetadataRequest{
		OrgID: org.ID,
		Path:  r.FormValue("path"),
		Value: r.FormValue("value"),
	}

	_, err = h.service.CreateMetadata(req)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/metadata", org.Slug))
	w.WriteHeader(http.StatusOK)
}

// EditMetadataForm handles GET /web/org/{org}/metadata/edit
func (h *Handler) EditMetadataForm(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		h.renderErrorBanner(w, "Metadata ID is required")
		return
	}

	metadata, err := h.service.GetMetadata(id)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("edit-metadata").Parse(editMetadataFormTemplate))
	tmpl.Execute(w, map[string]interface{}{"Org": org, "Metadata": metadata})
}

// UpdateMetadata handles PUT /web/org/{org}/metadata/update
func (h *Handler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderErrorBanner(w, "Failed to parse form")
		return
	}

	id := r.FormValue("id")
	value := r.FormValue("value")

	req := domain.UpdateMetadataRequest{Value: &value}

	_, err = h.service.UpdateMetadata(id, req)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/metadata", org.Slug))
	w.WriteHeader(http.StatusOK)
}

// DeleteMetadata handles DELETE /web/org/{org}/metadata/delete
func (h *Handler) DeleteMetadata(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		h.renderErrorBanner(w, "Metadata ID is required")
		return
	}

	if err := h.service.DeleteMetadata(id); err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/metadata", org.Slug))
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
		"Context": ctx,
		"Buckets": buckets,
	})
}

// NewBucketForm handles GET /web/org/{org}/projects/{project}/storage/buckets/new
func (h *Handler) NewBucketForm(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl := template.Must(template.New("new-bucket").Parse(newBucketFormTemplate))
	tmpl.Execute(w, map[string]interface{}{"Org": org, "Project": project})
}

// CreateBucket handles POST /web/org/{org}/projects/{project}/storage/buckets
func (h *Handler) CreateBucket(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderErrorBanner(w, "Failed to parse form")
		return
	}

	req := domain.CreateBucketRequest{
		Name: r.FormValue("name"),
	}

	_, err = h.service.CreateBucket(project.ID, req)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/projects/%s/storage", org.Slug, project.Slug))
	w.WriteHeader(http.StatusOK)
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
		h.renderErrorBanner(w, err.Error())
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
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
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderErrorBanner(w, "Failed to parse form")
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
		h.renderErrorBanner(w, err.Error())
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/web/org/%s/projects/%s/storage/%s", org.Slug, project.Slug, bucket.Name))
	w.WriteHeader(http.StatusOK)
}

// ViewObject handles GET /web/org/{org}/projects/{project}/storage/{bucket}/objects/{objid}
func (h *Handler) ViewObject(w http.ResponseWriter, r *http.Request) {
	org, project, err := h.resolveProject(r)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	objID := vars["objid"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	obj, err := h.service.GetObject(objID)
	if err != nil {
		h.renderErrorBanner(w, err.Error())
		return
	}

	// Decode content
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

// Templates

const errorTemplate = `<!DOCTYPE html>
<html>
<head><title>Error</title><link rel="icon" type="image/png" href="/web/static/logo.png"></head>
<body style="font-family: sans-serif; padding: 2rem;">
<h1>Error</h1>
<p>{{.Message}}</p>
<a href="/web">Go back</a>
</body>
</html>`

const errorBannerTemplate = `<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">{{.Message}}</div>`

const baseTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>NahCloud Console</title>
    <link rel="icon" type="image/png" href="/web/static/logo.png">
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        .modal { display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 1000; }
        .modal.active { display: flex; justify-content: center; align-items: flex-start; padding-top: 5vh; }
        .modal-content { background: white; border-radius: 0.5rem; max-width: 90%; max-height: 90vh; overflow: auto; }
    </style>
</head>
<body class="bg-gray-100">
    <div class="flex h-screen">
        <!-- Sidebar -->
        <div class="w-64 bg-gray-800 text-white flex-shrink-0">
            <div class="p-4 flex items-center gap-3">
                <img src="/web/static/logo.png" alt="NahCloud" class="h-8 w-8">
                <span class="text-xl font-bold">NahCloud</span>
            </div>

            <!-- Org/Project Selector -->
            <div class="px-4 py-2 border-b border-gray-700">
                {{if .Context.Org}}
                <div class="text-sm text-gray-400 mb-1">Organization</div>
                <select onchange="window.location.href='/web/org/' + this.value + '/projects'" class="w-full bg-gray-700 text-white px-2 py-1 rounded text-sm">
                    {{range .Context.Orgs}}
                    <option value="{{.Slug}}" {{if eq .Slug $.Context.Org.Slug}}selected{{end}}>{{.Name}}</option>
                    {{end}}
                </select>
                {{end}}

                {{if .Context.Project}}
                <div class="text-sm text-gray-400 mt-2 mb-1">Project</div>
                <select onchange="window.location.href='/web/org/{{.Context.Org.Slug}}/projects/' + this.value + '/instances'" class="w-full bg-gray-700 text-white px-2 py-1 rounded text-sm">
                    {{range .Context.Projects}}
                    <option value="{{.Slug}}" {{if eq .Slug $.Context.Project.Slug}}selected{{end}}>{{.Name}}</option>
                    {{end}}
                </select>
                {{end}}
            </div>

            <nav class="mt-4">
                {{if .Context.Org}}
                <a href="/web/org/{{.Context.Org.Slug}}/projects" class="block px-4 py-2 hover:bg-gray-700">Projects</a>
                {{end}}
                {{if .Context.Project}}
                <a href="/web/org/{{.Context.Org.Slug}}/projects/{{.Context.Project.Slug}}/instances" class="block px-4 py-2 hover:bg-gray-700">Instances</a>
                <a href="/web/org/{{.Context.Org.Slug}}/projects/{{.Context.Project.Slug}}/storage" class="block px-4 py-2 hover:bg-gray-700">Storage</a>
                {{end}}
                {{if .Context.Org}}
                <a href="/web/org/{{.Context.Org.Slug}}/metadata" class="block px-4 py-2 hover:bg-gray-700">Metadata</a>
                {{end}}
            </nav>
        </div>

        <!-- Main Content -->
        <div class="flex-1 overflow-auto">
            <div id="error-container"></div>
            <div class="p-6">
                {{block "content" .}}{{end}}
            </div>
        </div>
    </div>

    <!-- Modal Container -->
    <div id="modal" class="modal" onclick="if(event.target===this)closeModal()">
        <div id="modal-content" class="modal-content"></div>
    </div>

    <script>
        function openModal() { document.getElementById('modal').classList.add('active'); }
        function closeModal() { document.getElementById('modal').classList.remove('active'); }
        document.body.addEventListener('htmx:afterSwap', function(e) {
            if (e.target.id === 'modal-content') openModal();
        });
        document.body.addEventListener('htmx:responseError', function(e) {
            document.getElementById('error-container').innerHTML = '<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 mx-6 mt-6 rounded">Request failed: ' + e.detail.xhr.statusText + '</div>';
        });
    </script>
</body>
</html>`

const projectsTemplate = `{{define "content"}}
<div class="flex justify-between items-center mb-6">
    <h1 class="text-2xl font-bold">Projects</h1>
    <button hx-get="/web/org/{{.Context.Org.Slug}}/projects/new" hx-target="#modal-content" class="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600">New Project</button>
</div>

<div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="min-w-full">
        <thead class="bg-gray-50">
            <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Slug</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created At</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Updated At</th>
                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
            {{range .Projects}}
            <tr>
                <td class="px-6 py-4"><a href="/web/org/{{$.Context.Org.Slug}}/projects/{{.Slug}}/instances" class="text-blue-500 hover:underline">{{.Slug}}</a></td>
                <td class="px-6 py-4">{{.Name}}</td>
                <td class="px-6 py-4 text-sm text-gray-500">{{.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td class="px-6 py-4 text-sm text-gray-500">{{.UpdatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td class="px-6 py-4 text-right">
                    <button hx-get="/web/org/{{$.Context.Org.Slug}}/projects/{{.Slug}}/edit" hx-target="#modal-content" class="text-blue-500 hover:text-blue-700 mr-2">Edit</button>
                    <button hx-delete="/web/org/{{$.Context.Org.Slug}}/projects/{{.Slug}}" hx-confirm="Delete project {{.Name}}?" class="text-red-500 hover:text-red-700">Delete</button>
                </td>
            </tr>
            {{else}}
            <tr><td colspan="5" class="px-6 py-4 text-center text-gray-500">No projects found</td></tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}`

const newProjectFormTemplate = `<div class="p-6 w-96">
    <h2 class="text-xl font-bold mb-4">New Project</h2>
    <form hx-post="/web/org/{{.Org.Slug}}/projects" hx-swap="none">
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Slug</label>
            <input type="text" name="slug" required class="w-full border rounded px-3 py-2" placeholder="my-project" pattern="[a-z][a-z0-9-]*">
            <p class="text-xs text-gray-500 mt-1">Lowercase letters, numbers, and dashes. Must start with a letter.</p>
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input type="text" name="name" required class="w-full border rounded px-3 py-2" placeholder="My Project">
        </div>
        <div class="flex justify-end gap-2">
            <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Create</button>
        </div>
    </form>
</div>`

const editProjectFormTemplate = `<div class="p-6 w-96">
    <h2 class="text-xl font-bold mb-4">Edit Project</h2>
    <form hx-put="/web/org/{{.Org.Slug}}/projects/{{.Project.Slug}}" hx-swap="none">
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Slug</label>
            <input type="text" value="{{.Project.Slug}}" disabled class="w-full border rounded px-3 py-2 bg-gray-100">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input type="text" name="name" value="{{.Project.Name}}" required class="w-full border rounded px-3 py-2">
        </div>
        <div class="flex justify-end gap-2">
            <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Update</button>
        </div>
    </form>
</div>`

const instancesTemplate = `{{define "content"}}
<div class="flex justify-between items-center mb-6">
    <h1 class="text-2xl font-bold">Instances</h1>
    <button hx-get="/web/org/{{.Context.Org.Slug}}/projects/{{.Context.Project.Slug}}/instances/new" hx-target="#modal-content" class="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600">New Instance</button>
</div>

<div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="min-w-full">
        <thead class="bg-gray-50">
            <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Region</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">CPU</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Memory</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
            {{range .Instances}}
            <tr>
                <td class="px-6 py-4">{{.Name}}</td>
                <td class="px-6 py-4">{{.Region}}</td>
                <td class="px-6 py-4">{{.CPU}}</td>
                <td class="px-6 py-4">{{.MemoryMB}} MB</td>
                <td class="px-6 py-4">
                    <span class="px-2 py-1 text-xs rounded {{if eq .Status "running"}}bg-green-100 text-green-800{{else}}bg-gray-100 text-gray-800{{end}}">{{.Status}}</span>
                </td>
                <td class="px-6 py-4 text-right">
                    <button hx-get="/web/org/{{$.Context.Org.Slug}}/projects/{{$.Context.Project.Slug}}/instances/{{.ID}}/edit" hx-target="#modal-content" class="text-blue-500 hover:text-blue-700 mr-2">Edit</button>
                    <button hx-delete="/web/org/{{$.Context.Org.Slug}}/projects/{{$.Context.Project.Slug}}/instances/{{.ID}}" hx-confirm="Delete instance {{.Name}}?" class="text-red-500 hover:text-red-700">Delete</button>
                </td>
            </tr>
            {{else}}
            <tr><td colspan="6" class="px-6 py-4 text-center text-gray-500">No instances found</td></tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}`

const newInstanceFormTemplate = `<div class="p-6 w-96">
    <h2 class="text-xl font-bold mb-4">New Instance</h2>
    <form hx-post="/web/org/{{.Org.Slug}}/projects/{{.Project.Slug}}/instances" hx-swap="none">
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input type="text" name="name" required class="w-full border rounded px-3 py-2">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Region</label>
            <select name="region" required class="w-full border rounded px-3 py-2">
                {{range .Regions}}<option value="{{.}}">{{.}}</option>{{end}}
            </select>
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">CPU Cores</label>
            <input type="number" name="cpu" value="1" min="1" max="64" required class="w-full border rounded px-3 py-2">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Memory (MB)</label>
            <input type="number" name="memory_mb" value="1024" min="1" required class="w-full border rounded px-3 py-2">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Image</label>
            <input type="text" name="image" required class="w-full border rounded px-3 py-2" placeholder="ubuntu-22.04">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Status</label>
            <select name="status" class="w-full border rounded px-3 py-2">
                <option value="running">Running</option>
                <option value="stopped">Stopped</option>
            </select>
        </div>
        <div class="flex justify-end gap-2">
            <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Create</button>
        </div>
    </form>
</div>`

const editInstanceFormTemplate = `<div class="p-6 w-96">
    <h2 class="text-xl font-bold mb-4">Edit Instance</h2>
    <form hx-put="/web/org/{{.Org.Slug}}/projects/{{.Project.Slug}}/instances/{{.Instance.ID}}" hx-swap="none">
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input type="text" name="name" value="{{.Instance.Name}}" required class="w-full border rounded px-3 py-2">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Region</label>
            <input type="text" value="{{.Instance.Region}}" disabled class="w-full border rounded px-3 py-2 bg-gray-100">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">CPU Cores</label>
            <input type="number" name="cpu" value="{{.Instance.CPU}}" min="1" max="64" required class="w-full border rounded px-3 py-2">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Memory (MB)</label>
            <input type="number" name="memory_mb" value="{{.Instance.MemoryMB}}" min="1" required class="w-full border rounded px-3 py-2">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Image</label>
            <input type="text" value="{{.Instance.Image}}" disabled class="w-full border rounded px-3 py-2 bg-gray-100">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Status</label>
            <select name="status" class="w-full border rounded px-3 py-2">
                <option value="running" {{if eq .Instance.Status "running"}}selected{{end}}>Running</option>
                <option value="stopped" {{if eq .Instance.Status "stopped"}}selected{{end}}>Stopped</option>
            </select>
        </div>
        <div class="flex justify-end gap-2">
            <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Update</button>
        </div>
    </form>
</div>`

const metadataTemplate = `{{define "content"}}
<div class="flex justify-between items-center mb-6">
    <h1 class="text-2xl font-bold">Metadata</h1>
    <button hx-get="/web/org/{{.Context.Org.Slug}}/metadata/new" hx-target="#modal-content" class="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600">New Entry</button>
</div>

<div class="mb-4">
    <form method="get" class="flex gap-2">
        <input type="text" name="prefix" value="{{.Prefix}}" placeholder="Filter by prefix..." class="border rounded px-3 py-2 flex-1">
        <button type="submit" class="bg-gray-500 text-white px-4 py-2 rounded hover:bg-gray-600">Filter</button>
    </form>
</div>

<div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="min-w-full">
        <thead class="bg-gray-50">
            <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Path</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Value</th>
                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
            {{range .Metadata}}
            <tr>
                <td class="px-6 py-4 font-mono text-sm">{{.Path}}</td>
                <td class="px-6 py-4 text-sm max-w-md truncate">{{.Value}}</td>
                <td class="px-6 py-4 text-right">
                    <button hx-get="/web/org/{{$.Context.Org.Slug}}/metadata/edit?id={{.ID}}" hx-target="#modal-content" class="text-blue-500 hover:text-blue-700 mr-2">Edit</button>
                    <button hx-delete="/web/org/{{$.Context.Org.Slug}}/metadata/delete?id={{.ID}}" hx-confirm="Delete metadata {{.Path}}?" class="text-red-500 hover:text-red-700">Delete</button>
                </td>
            </tr>
            {{else}}
            <tr><td colspan="3" class="px-6 py-4 text-center text-gray-500">No metadata found</td></tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}`

const newMetadataFormTemplate = `<div class="p-6 w-96">
    <h2 class="text-xl font-bold mb-4">New Metadata Entry</h2>
    <form hx-post="/web/org/{{.Org.Slug}}/metadata" hx-swap="none">
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Path</label>
            <input type="text" name="path" required class="w-full border rounded px-3 py-2" placeholder="config/setting">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Value</label>
            <textarea name="value" required class="w-full border rounded px-3 py-2" rows="4"></textarea>
        </div>
        <div class="flex justify-end gap-2">
            <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Create</button>
        </div>
    </form>
</div>`

const editMetadataFormTemplate = `<div class="p-6 w-96">
    <h2 class="text-xl font-bold mb-4">Edit Metadata Entry</h2>
    <form hx-put="/web/org/{{.Org.Slug}}/metadata/update" hx-swap="none">
        <input type="hidden" name="id" value="{{.Metadata.ID}}">
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Path</label>
            <input type="text" value="{{.Metadata.Path}}" disabled class="w-full border rounded px-3 py-2 bg-gray-100">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Value</label>
            <textarea name="value" required class="w-full border rounded px-3 py-2" rows="4">{{.Metadata.Value}}</textarea>
        </div>
        <div class="flex justify-end gap-2">
            <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Update</button>
        </div>
    </form>
</div>`

const storageTemplate = `{{define "content"}}
<div class="flex justify-between items-center mb-6">
    <h1 class="text-2xl font-bold">Storage Buckets</h1>
    <button hx-get="/web/org/{{.Context.Org.Slug}}/projects/{{.Context.Project.Slug}}/storage/buckets/new" hx-target="#modal-content" class="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600">New Bucket</button>
</div>

<div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="min-w-full">
        <thead class="bg-gray-50">
            <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created At</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Updated At</th>
            </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
            {{range .Buckets}}
            <tr>
                <td class="px-6 py-4"><a href="/web/org/{{$.Context.Org.Slug}}/projects/{{$.Context.Project.Slug}}/storage/{{.Name}}" class="text-blue-500 hover:underline">{{.Name}}</a></td>
                <td class="px-6 py-4 text-sm text-gray-500">{{.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td class="px-6 py-4 text-sm text-gray-500">{{.UpdatedAt.Format "2006-01-02 15:04:05"}}</td>
            </tr>
            {{else}}
            <tr><td colspan="3" class="px-6 py-4 text-center text-gray-500">No buckets found</td></tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}`

const newBucketFormTemplate = `<div class="p-6 w-96">
    <h2 class="text-xl font-bold mb-4">New Bucket</h2>
    <form hx-post="/web/org/{{.Org.Slug}}/projects/{{.Project.Slug}}/storage/buckets" hx-swap="none">
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input type="text" name="name" required class="w-full border rounded px-3 py-2" pattern="[a-zA-Z0-9_-]+">
            <p class="text-xs text-gray-500 mt-1">Alphanumeric, dashes, and underscores only.</p>
        </div>
        <div class="flex justify-end gap-2">
            <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Create</button>
        </div>
    </form>
</div>`

const bucketObjectsTemplate = `{{define "content"}}
<div class="mb-4">
    <a href="/web/org/{{.Context.Org.Slug}}/projects/{{.Context.Project.Slug}}/storage" class="text-blue-500 hover:underline">&larr; Back to Buckets</a>
</div>

<div class="flex justify-between items-center mb-6">
    <h1 class="text-2xl font-bold">{{.Bucket.Name}}</h1>
    <button hx-get="/web/org/{{.Context.Org.Slug}}/projects/{{.Context.Project.Slug}}/storage/{{.Bucket.Name}}/objects/new" hx-target="#modal-content" class="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600">New Object</button>
</div>

<div class="mb-4">
    <form method="get" class="flex gap-2">
        <input type="text" name="prefix" value="{{.Prefix}}" placeholder="Filter by prefix..." class="border rounded px-3 py-2 flex-1">
        <button type="submit" class="bg-gray-500 text-white px-4 py-2 rounded hover:bg-gray-600">Filter</button>
    </form>
</div>

<div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="min-w-full">
        <thead class="bg-gray-50">
            <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Path</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Updated At</th>
                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
        </thead>
        <tbody class="divide-y divide-gray-200">
            {{range .Objects}}
            <tr>
                <td class="px-6 py-4 font-mono text-sm">{{.Path}}</td>
                <td class="px-6 py-4 text-sm text-gray-500">{{.UpdatedAt.Format "2006-01-02 15:04:05"}}</td>
                <td class="px-6 py-4 text-right">
                    <button hx-get="/web/org/{{$.Context.Org.Slug}}/projects/{{$.Context.Project.Slug}}/storage/{{$.Bucket.Name}}/objects/{{.ID}}" hx-target="#modal-content" class="text-blue-500 hover:text-blue-700">View</button>
                </td>
            </tr>
            {{else}}
            <tr><td colspan="3" class="px-6 py-4 text-center text-gray-500">No objects found</td></tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}`

const newObjectFormTemplate = `<div class="p-6 w-96">
    <h2 class="text-xl font-bold mb-4">New Object</h2>
    <form hx-post="/web/org/{{.Org.Slug}}/projects/{{.Project.Slug}}/storage/{{.Bucket.Name}}/objects" hx-swap="none">
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Path</label>
            <input type="text" name="path" required class="w-full border rounded px-3 py-2" placeholder="path/to/file.txt">
        </div>
        <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Content</label>
            <textarea name="content" required class="w-full border rounded px-3 py-2 font-mono text-sm" rows="6"></textarea>
        </div>
        <div class="flex justify-end gap-2">
            <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Cancel</button>
            <button type="submit" class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Create</button>
        </div>
    </form>
</div>`

const viewObjectTemplate = `<div class="p-6 w-[600px]">
    <h2 class="text-xl font-bold mb-4">{{.Object.Path}}</h2>
    <div class="mb-4 text-sm text-gray-500">Size: {{.Size}} bytes</div>
    <div class="bg-gray-100 p-4 rounded font-mono text-sm whitespace-pre-wrap overflow-auto max-h-96">{{.DecodedContent}}</div>
    <div class="mt-4 flex justify-end">
        <button type="button" onclick="closeModal()" class="px-4 py-2 border rounded hover:bg-gray-50">Close</button>
    </div>
</div>`
