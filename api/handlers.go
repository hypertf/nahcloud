package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/service"
)

// Handler handles HTTP requests
type Handler struct {
	service *service.Service
}

// NewHandler creates a new handler
func NewHandler(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	if domain.IsNotFound(err) {
		status = http.StatusNotFound
		message = err.Error()
	} else if domain.IsAlreadyExists(err) {
		status = http.StatusConflict
		message = err.Error()
	} else if domain.IsInvalidInput(err) {
		status = http.StatusBadRequest
		message = err.Error()
	} else if domain.IsForeignKeyViolation(err) {
		status = http.StatusBadRequest
		message = err.Error()
	} else if domain.IsUnauthorized(err) {
		status = http.StatusUnauthorized
		message = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Helper functions to resolve org and project from URL path

// resolveOrg gets org from URL and verifies the authenticated org matches
func (h *Handler) resolveOrg(r *http.Request) (*domain.Organization, error) {
	vars := mux.Vars(r)
	orgSlug := vars["org"]

	if orgSlug == "" {
		return nil, domain.InvalidInputError("organization slug is required", nil)
	}

	// Get the org from the URL
	org, err := h.service.GetOrganizationBySlug(orgSlug)
	if err != nil {
		return nil, err
	}

	// Verify the authenticated org (from token) matches the requested org
	authOrg := OrgFromContext(r.Context())
	if authOrg != nil && authOrg.ID != org.ID {
		return nil, domain.UnauthorizedError("token does not have access to this organization")
	}

	return org, nil
}

// resolveProject gets org and project from URL and returns the project
func (h *Handler) resolveProject(r *http.Request) (*domain.Project, error) {
	org, err := h.resolveOrg(r)
	if err != nil {
		return nil, err
	}

	vars := mux.Vars(r)
	projectSlug := vars["project"]

	if projectSlug == "" {
		return nil, domain.InvalidInputError("project slug is required", nil)
	}

	return h.service.GetProjectBySlug(org.ID, projectSlug)
}

// decodeJSON decodes JSON from the request body into the given value
func (h *Handler) decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return domain.InvalidInputError("invalid JSON", nil)
	}
	return nil
}

// Organization handlers

// CreateOrganization handles POST /v1/orgs
func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateOrganizationRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	org, err := h.service.CreateOrganization(req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, org)
}

// GetOrganization handles GET /v1/orgs/{org}
func (h *Handler) GetOrganization(w http.ResponseWriter, r *http.Request) {


	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, org)
}

// TODO: Add admin controls for ListOrganizations, UpdateOrganization, DeleteOrganization

// API Key handlers

// CreateAPIKey handles POST /v1/orgs/{org}/api-keys
func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req domain.CreateAPIKeyRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	apiKey, err := h.service.CreateAPIKey(org.ID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, apiKey)
}

// ListAPIKeys handles GET /v1/orgs/{org}/api-keys
func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	keys, err := h.service.ListAPIKeys(org.ID)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, keys)
}

// DeleteAPIKey handles DELETE /v1/orgs/{org}/api-keys/{key_id}
func (h *Handler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	keyID := vars["key_id"]

	if err := h.service.DeleteAPIKey(org.ID, keyID); err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Project handlers

// CreateProject handles POST /v1/orgs/{org}/projects
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req domain.CreateProjectRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	project, err := h.service.CreateProject(org.ID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, project)
}

// GetProject handles GET /v1/orgs/{org}/projects/{project}
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, project)
}

// ListProjects handles GET /v1/orgs/{org}/projects
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {


	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	opts := domain.ProjectListOptions{
		OrgID: org.ID,
		Name:  r.URL.Query().Get("name"),
	}

	projects, err := h.service.ListProjects(opts)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, projects)
}

// UpdateProject handles PATCH /v1/orgs/{org}/projects/{project}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req domain.UpdateProjectRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	updated, err := h.service.UpdateProject(project.ID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, updated)
}

// DeleteProject handles DELETE /v1/orgs/{org}/projects/{project}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	if err := h.service.DeleteProject(project.ID); err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Instance handlers

// CreateInstance handles POST /v1/orgs/{org}/projects/{project}/instances
func (h *Handler) CreateInstance(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req domain.CreateInstanceRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	// Force the project ID from the URL
	req.ProjectID = project.ID

	instance, err := h.service.CreateInstance(req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, instance)
}

// GetInstance handles GET /v1/orgs/{org}/projects/{project}/instances/{id}
func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	instance, err := h.service.GetInstance(id)
	if err != nil {
		h.writeError(w, err)
		return
	}

	// Ensure instance belongs to the project
	if instance.ProjectID != project.ID {
		h.writeError(w, domain.NotFoundError("instance", id))
		return
	}

	h.writeJSON(w, http.StatusOK, instance)
}

// ListInstances handles GET /v1/orgs/{org}/projects/{project}/instances
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	opts := domain.InstanceListOptions{
		ProjectID: project.ID,
		Name:      r.URL.Query().Get("name"),
		Region:    r.URL.Query().Get("region"),
		Status:    r.URL.Query().Get("status"),
	}

	instances, err := h.service.ListInstances(opts)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, instances)
}

// UpdateInstance handles PATCH /v1/orgs/{org}/projects/{project}/instances/{id}
func (h *Handler) UpdateInstance(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Verify instance belongs to project
	instance, err := h.service.GetInstance(id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if instance.ProjectID != project.ID {
		h.writeError(w, domain.NotFoundError("instance", id))
		return
	}

	var req domain.UpdateInstanceRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	updated, err := h.service.UpdateInstance(id, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, updated)
}

// DeleteInstance handles DELETE /v1/orgs/{org}/projects/{project}/instances/{id}
func (h *Handler) DeleteInstance(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Verify instance belongs to project
	instance, err := h.service.GetInstance(id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if instance.ProjectID != project.ID {
		h.writeError(w, domain.NotFoundError("instance", id))
		return
	}

	if err := h.service.DeleteInstance(id); err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Metadata handlers

// CreateMetadata handles POST /v1/orgs/{org}/metadata
func (h *Handler) CreateMetadata(w http.ResponseWriter, r *http.Request) {


	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req domain.CreateMetadataRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	// Force the org ID from the URL
	req.OrgID = org.ID

	metadata, err := h.service.CreateMetadata(req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, metadata)
}

// GetMetadata handles GET /v1/orgs/{org}/metadata/{id}
func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {


	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	metadata, err := h.service.GetMetadata(id)
	if err != nil {
		h.writeError(w, err)
		return
	}

	// Ensure metadata belongs to the org
	if metadata.OrgID != org.ID {
		h.writeError(w, domain.NotFoundError("metadata", id))
		return
	}

	h.writeJSON(w, http.StatusOK, metadata)
}

// ListMetadata handles GET /v1/orgs/{org}/metadata
func (h *Handler) ListMetadata(w http.ResponseWriter, r *http.Request) {


	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	opts := domain.MetadataListOptions{
		OrgID:  org.ID,
		Prefix: r.URL.Query().Get("prefix"),
	}

	metadata, err := h.service.ListMetadata(opts)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, metadata)
}

// UpdateMetadata handles PATCH /v1/orgs/{org}/metadata/{id}
func (h *Handler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {


	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Verify metadata belongs to org
	metadata, err := h.service.GetMetadata(id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if metadata.OrgID != org.ID {
		h.writeError(w, domain.NotFoundError("metadata", id))
		return
	}

	var req domain.UpdateMetadataRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	updated, err := h.service.UpdateMetadata(id, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, updated)
}

// DeleteMetadata handles DELETE /v1/orgs/{org}/metadata/{id}
func (h *Handler) DeleteMetadata(w http.ResponseWriter, r *http.Request) {


	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	// Verify metadata belongs to org
	metadata, err := h.service.GetMetadata(id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if metadata.OrgID != org.ID {
		h.writeError(w, domain.NotFoundError("metadata", id))
		return
	}

	if err := h.service.DeleteMetadata(id); err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Bucket handlers

// CreateBucket handles POST /v1/orgs/{org}/projects/{project}/buckets
func (h *Handler) CreateBucket(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req domain.CreateBucketRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	bucket, err := h.service.CreateBucket(project.ID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, bucket)
}

// GetBucket handles GET /v1/orgs/{org}/projects/{project}/buckets/{bucket}
func (h *Handler) GetBucket(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, bucket)
}

// ListBuckets handles GET /v1/orgs/{org}/projects/{project}/buckets
func (h *Handler) ListBuckets(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	opts := domain.BucketListOptions{
		ProjectID: project.ID,
		Name:      r.URL.Query().Get("name"),
	}

	buckets, err := h.service.ListBuckets(opts)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, buckets)
}

// UpdateBucket handles PATCH /v1/orgs/{org}/projects/{project}/buckets/{bucket}
func (h *Handler) UpdateBucket(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req domain.UpdateBucketRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	updated, err := h.service.UpdateBucket(bucket.ID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, updated)
}

// DeleteBucket handles DELETE /v1/orgs/{org}/projects/{project}/buckets/{bucket}
func (h *Handler) DeleteBucket(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	if err := h.service.DeleteBucket(bucket.ID); err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Object handlers

// CreateObject handles POST /v1/orgs/{org}/projects/{project}/buckets/{bucket}/objects
func (h *Handler) CreateObject(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var req domain.CreateObjectRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	// Force the bucket from the URL
	req.BucketID = bucket.ID

	obj, err := h.service.CreateObject(req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, obj)
}

// GetObject handles GET /v1/orgs/{org}/projects/{project}/buckets/{bucket}/objects/{id}
func (h *Handler) GetObject(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	id := vars["id"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	obj, err := h.service.GetObject(id)
	if err != nil {
		h.writeError(w, err)
		return
	}

	// Enforce object belongs to the requested bucket
	if obj.BucketID != bucket.ID {
		h.writeError(w, domain.NotFoundError("object", id))
		return
	}

	h.writeJSON(w, http.StatusOK, obj)
}

// ListObjects handles GET /v1/orgs/{org}/projects/{project}/buckets/{bucket}/objects
func (h *Handler) ListObjects(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	opts := domain.ObjectListOptions{
		BucketID: bucket.ID,
		Prefix:   r.URL.Query().Get("prefix"),
	}

	objects, err := h.service.ListObjects(opts)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, objects)
}

// UpdateObject handles PATCH /v1/orgs/{org}/projects/{project}/buckets/{bucket}/objects/{id}
func (h *Handler) UpdateObject(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	id := vars["id"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	// Verify object belongs to bucket
	obj, err := h.service.GetObject(id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if obj.BucketID != bucket.ID {
		h.writeError(w, domain.NotFoundError("object", id))
		return
	}

	var req domain.UpdateObjectRequest
	if err := h.decodeJSON(r, &req); err != nil {
		h.writeError(w, err)
		return
	}

	updated, err := h.service.UpdateObject(id, req)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, updated)
}

// DeleteObject handles DELETE /v1/orgs/{org}/projects/{project}/buckets/{bucket}/objects/{id}
func (h *Handler) DeleteObject(w http.ResponseWriter, r *http.Request) {


	project, err := h.resolveProject(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucket"]
	id := vars["id"]

	bucket, err := h.service.GetBucketByName(project.ID, bucketName)
	if err != nil {
		h.writeError(w, err)
		return
	}

	// Verify object belongs to bucket
	obj, err := h.service.GetObject(id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if obj.BucketID != bucket.ID {
		h.writeError(w, domain.NotFoundError("object", id))
		return
	}

	if err := h.service.DeleteObject(id); err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
