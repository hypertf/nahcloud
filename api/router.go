package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/service"
	"github.com/hypertf/nahcloud/web"
)

var startTime = time.Now()

const maxRequestBodyBytes = 16 << 20

// BuildInfo contains server build and runtime information
type BuildInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Uptime    string `json:"uptime"`
}

// SetupRouter creates and configures the HTTP router
func SetupRouter(handler *Handler, svc *service.Service, version string) *mux.Router {
	router := mux.NewRouter()

	// Build info endpoint (public)
	router.HandleFunc("/buildz", func(w http.ResponseWriter, r *http.Request) {
		info := BuildInfo{
			Version:   version,
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			Uptime:    time.Since(startTime).Round(time.Second).String(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}).Methods("GET")

	// Static assets (no auth required)
	router.HandleFunc("/static/logo.png", web.NewHandler(svc).ServeLogo).Methods("GET")

	// Public org permalinks (no auth required)
	webHandler := web.NewHandler(svc)
	router.HandleFunc("/o/{org}", webHandler.OpenPermalink).Methods("GET")

	// Web console routes with session-based auth
	webRouter := router.PathPrefix("").Subrouter()
	webRouter.Use(WebSessionMiddleware(svc))

	// Default landing page and current project routes
	webRouter.HandleFunc("", webHandler.ListCurrentInstances).Methods("GET")
	webRouter.HandleFunc("/", webHandler.ListCurrentInstances).Methods("GET")
	webRouter.HandleFunc("/instances", webHandler.ListCurrentInstances).Methods("GET")
	webRouter.HandleFunc("/storage", webHandler.ListCurrentStorage).Methods("GET")
	webRouter.HandleFunc("/cloud", webHandler.ListCurrentCloud).Methods("GET")

	// Settings
	webRouter.HandleFunc("/settings", webHandler.Settings).Methods("GET")
	webRouter.HandleFunc("/settings/organization", webHandler.UpdateOrganization).Methods("PUT")
	webRouter.HandleFunc("/settings/api-keys", webHandler.CreateAPIKey).Methods("POST")
	webRouter.HandleFunc("/settings/api-keys/{id}", webHandler.DeleteAPIKey).Methods("DELETE")
	webRouter.HandleFunc("/settings/reset", webHandler.ResetOrganization).Methods("POST")

	// Projects list
	webRouter.HandleFunc("/projects", webHandler.ListProjects).Methods("GET")
	webRouter.HandleFunc("/projects", webHandler.CreateProject).Methods("POST")
	webRouter.HandleFunc("/projects/new", webHandler.NewProjectForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}", webHandler.SelectProject).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/edit", webHandler.EditProjectForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}", webHandler.UpdateProject).Methods("PUT")
	webRouter.HandleFunc("/projects/{project}", webHandler.DeleteProject).Methods("DELETE")

	// Instances (scoped to project)
	webRouter.HandleFunc("/projects/{project}/instances", webHandler.ListInstances).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/instances", webHandler.CreateInstance).Methods("POST")
	webRouter.HandleFunc("/projects/{project}/instances/new", webHandler.NewInstanceForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/instances/{id}/edit", webHandler.EditInstanceForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/instances/{id}", webHandler.UpdateInstance).Methods("PUT")
	webRouter.HandleFunc("/projects/{project}/instances/{id}", webHandler.DeleteInstance).Methods("DELETE")

	// Proposed cloud graph console (project-scoped preview contract)
	webRouter.HandleFunc("/projects/{project}/cloud", webHandler.ListCloud).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/cloud/new", webHandler.NewGraphResourceForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/cloud/resources", webHandler.CreateGraphResource).Methods("POST")
	webRouter.HandleFunc("/projects/{project}/cloud/{kind}/{id}", webHandler.DeleteGraphResource).Methods("DELETE")
	webRouter.HandleFunc("/projects/{project}/cloud/faults/{fault}", webHandler.SetGraphFault).Methods("POST")

	// Storage (scoped to project)
	webRouter.HandleFunc("/projects/{project}/storage", webHandler.ListStorage).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/storage/buckets/new", webHandler.NewBucketForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/storage/buckets", webHandler.CreateBucket).Methods("POST")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}", webHandler.ListBucketObjects).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}/edit", webHandler.EditBucketForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}", webHandler.UpdateBucket).Methods("PUT")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}", webHandler.DeleteBucket).Methods("DELETE")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}/objects/new", webHandler.NewObjectForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}/objects", webHandler.CreateObject).Methods("POST")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}/objects/{objid}", webHandler.ViewObject).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}/objects/{objid}/edit", webHandler.EditObjectForm).Methods("GET")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}/objects/{objid}", webHandler.UpdateObject).Methods("PUT")
	webRouter.HandleFunc("/projects/{project}/storage/{bucket}/objects/{objid}", webHandler.DeleteObject).Methods("DELETE")

	// Metadata (scoped to org - from context)
	webRouter.HandleFunc("/metadata", webHandler.ListMetadata).Methods("GET")
	webRouter.HandleFunc("/metadata", webHandler.CreateMetadata).Methods("POST")
	webRouter.HandleFunc("/metadata/new", webHandler.NewMetadataForm).Methods("GET")
	webRouter.HandleFunc("/metadata/edit", webHandler.EditMetadataForm).Methods("GET")
	webRouter.HandleFunc("/metadata/update", webHandler.UpdateMetadata).Methods("PUT")
	webRouter.HandleFunc("/metadata/delete", webHandler.DeleteMetadata).Methods("DELETE")

	// API prefix
	api := router.PathPrefix("/v1").Subrouter()

	// Public routes (no auth required)
	api.HandleFunc("/orgs", handler.CreateOrganization).Methods("POST")

	// Authenticated API routes require an organization API token.
	authAPI := api.PathPrefix("").Subrouter()
	authAPI.Use(AuthMiddleware(svc))

	// Organization routes (authenticated - returns current org)
	authAPI.HandleFunc("/org", handler.GetOrganization).Methods("GET")
	authAPI.HandleFunc("/org", handler.UpdateOrganization).Methods("PATCH")
	authAPI.HandleFunc("/org/reset", handler.ResetOrganization).Methods("POST")

	// API Key routes (scoped to current org)
	authAPI.HandleFunc("/api-keys", handler.CreateAPIKey).Methods("POST")
	authAPI.HandleFunc("/api-keys", handler.ListAPIKeys).Methods("GET")
	authAPI.HandleFunc("/api-keys/{key_id}", handler.GetAPIKey).Methods("GET")
	authAPI.HandleFunc("/api-keys/{key_id}", handler.DeleteAPIKey).Methods("DELETE")

	// Project routes (scoped to current org)
	authAPI.HandleFunc("/projects", handler.CreateProject).Methods("POST")
	authAPI.HandleFunc("/projects", handler.ListProjects).Methods("GET")
	authAPI.HandleFunc("/projects/{project}", handler.GetProject).Methods("GET")
	authAPI.HandleFunc("/projects/{project}", handler.UpdateProject).Methods("PATCH")
	authAPI.HandleFunc("/projects/{project}", handler.DeleteProject).Methods("DELETE")

	// Instance routes (scoped to project)
	authAPI.HandleFunc("/projects/{project}/instances", handler.CreateInstance).Methods("POST")
	authAPI.HandleFunc("/projects/{project}/instances", handler.ListInstances).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/instances/{id}", handler.GetInstance).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/instances/{id}", handler.UpdateInstance).Methods("PATCH")
	authAPI.HandleFunc("/projects/{project}/instances/{id}", handler.DeleteInstance).Methods("DELETE")

	// Bucket routes (scoped to project)
	authAPI.HandleFunc("/projects/{project}/buckets", handler.CreateBucket).Methods("POST")
	authAPI.HandleFunc("/projects/{project}/buckets", handler.ListBuckets).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/buckets/{bucket}", handler.GetBucket).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/buckets/{bucket}", handler.UpdateBucket).Methods("PATCH")
	authAPI.HandleFunc("/projects/{project}/buckets/{bucket}", handler.DeleteBucket).Methods("DELETE")
	authAPI.HandleFunc("/projects/{project}/buckets-by-id/{bucket_id}", handler.GetBucket).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/buckets-by-id/{bucket_id}", handler.UpdateBucket).Methods("PATCH")
	authAPI.HandleFunc("/projects/{project}/buckets-by-id/{bucket_id}", handler.DeleteBucket).Methods("DELETE")

	// Object routes (scoped to bucket)
	authAPI.HandleFunc("/projects/{project}/buckets/{bucket}/objects", handler.CreateObject).Methods("POST")
	authAPI.HandleFunc("/projects/{project}/buckets/{bucket}/objects", handler.ListObjects).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/buckets/{bucket}/objects/{id}", handler.GetObject).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/buckets/{bucket}/objects/{id}", handler.UpdateObject).Methods("PATCH")
	authAPI.HandleFunc("/projects/{project}/buckets/{bucket}/objects/{id}", handler.DeleteObject).Methods("DELETE")
	authAPI.HandleFunc("/projects/{project}/buckets-by-id/{bucket_id}/objects", handler.CreateObject).Methods("POST")
	authAPI.HandleFunc("/projects/{project}/buckets-by-id/{bucket_id}/objects", handler.ListObjects).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/buckets-by-id/{bucket_id}/objects/{id}", handler.GetObject).Methods("GET")
	authAPI.HandleFunc("/projects/{project}/buckets-by-id/{bucket_id}/objects/{id}", handler.UpdateObject).Methods("PATCH")
	authAPI.HandleFunc("/projects/{project}/buckets-by-id/{bucket_id}/objects/{id}", handler.DeleteObject).Methods("DELETE")

	// Metadata routes (scoped to current org)
	authAPI.HandleFunc("/metadata", handler.CreateMetadata).Methods("POST")
	authAPI.HandleFunc("/metadata", handler.ListMetadata).Methods("GET").Queries("prefix", "")
	authAPI.HandleFunc("/metadata", handler.ListMetadata).Methods("GET")
	authAPI.HandleFunc("/metadata/{id}", handler.GetMetadata).Methods("GET")
	authAPI.HandleFunc("/metadata/{id}", handler.UpdateMetadata).Methods("PATCH")
	authAPI.HandleFunc("/metadata/{id}", handler.DeleteMetadata).Methods("DELETE")

	// Terraform HTTP backend routes (scoped to the authenticated organization)
	authAPI.HandleFunc("/tfstate/{id}", handler.TFStateGet).Methods("GET")
	authAPI.HandleFunc("/tfstate/{id}", handler.TFStatePost).Methods("POST")
	authAPI.HandleFunc("/tfstate/{id}", handler.TFStateDelete).Methods("DELETE")
	authAPI.HandleFunc("/tfstate/{id}", handler.TFStateLock).Methods("LOCK")
	authAPI.HandleFunc("/tfstate/{id}", handler.TFStateUnlock).Methods("UNLOCK")

	// Add CORS middleware for development
	router.Use(corsMiddleware)
	router.Use(limitRequestBodyMiddleware)

	// Add logging middleware
	router.Use(loggingMiddleware)

	return router
}

func limitRequestBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, LOCK, UNLOCK, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware adds basic request logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Add proper structured logging here
		// For now, we'll let the main server handle logging
		next.ServeHTTP(w, r)
	})
}
