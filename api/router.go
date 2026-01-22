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

	// Web console routes (no API auth - web has its own session handling)
	webHandler := web.NewHandler(svc)
	webRouter := router.PathPrefix("/web").Subrouter()

	// Static assets
	webRouter.HandleFunc("/static/logo.png", webHandler.ServeLogo).Methods("GET")

	// Dashboard (redirects to default org)
	webRouter.HandleFunc("", webHandler.Dashboard).Methods("GET")
	webRouter.HandleFunc("/", webHandler.Dashboard).Methods("GET")

	// Org-scoped web routes
	// Projects list
	webRouter.HandleFunc("/org/{org}/projects", webHandler.ListProjects).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects", webHandler.CreateProject).Methods("POST")
	webRouter.HandleFunc("/org/{org}/projects/new", webHandler.NewProjectForm).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}/edit", webHandler.EditProjectForm).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}", webHandler.UpdateProject).Methods("PUT")
	webRouter.HandleFunc("/org/{org}/projects/{project}", webHandler.DeleteProject).Methods("DELETE")

	// Instances (scoped to project)
	webRouter.HandleFunc("/org/{org}/projects/{project}/instances", webHandler.ListInstances).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}/instances", webHandler.CreateInstance).Methods("POST")
	webRouter.HandleFunc("/org/{org}/projects/{project}/instances/new", webHandler.NewInstanceForm).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}/instances/{id}/edit", webHandler.EditInstanceForm).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}/instances/{id}", webHandler.UpdateInstance).Methods("PUT")
	webRouter.HandleFunc("/org/{org}/projects/{project}/instances/{id}", webHandler.DeleteInstance).Methods("DELETE")

	// Storage (scoped to project)
	webRouter.HandleFunc("/org/{org}/projects/{project}/storage", webHandler.ListStorage).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}/storage/buckets/new", webHandler.NewBucketForm).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}/storage/buckets", webHandler.CreateBucket).Methods("POST")
	webRouter.HandleFunc("/org/{org}/projects/{project}/storage/{bucket}", webHandler.ListBucketObjects).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}/storage/{bucket}/objects/new", webHandler.NewObjectForm).Methods("GET")
	webRouter.HandleFunc("/org/{org}/projects/{project}/storage/{bucket}/objects", webHandler.CreateObject).Methods("POST")
	webRouter.HandleFunc("/org/{org}/projects/{project}/storage/{bucket}/objects/{objid}", webHandler.ViewObject).Methods("GET")

	// Metadata (scoped to org)
	webRouter.HandleFunc("/org/{org}/metadata", webHandler.ListMetadata).Methods("GET")
	webRouter.HandleFunc("/org/{org}/metadata", webHandler.CreateMetadata).Methods("POST")
	webRouter.HandleFunc("/org/{org}/metadata/new", webHandler.NewMetadataForm).Methods("GET")
	webRouter.HandleFunc("/org/{org}/metadata/edit", webHandler.EditMetadataForm).Methods("GET")
	webRouter.HandleFunc("/org/{org}/metadata/update", webHandler.UpdateMetadata).Methods("PUT")
	webRouter.HandleFunc("/org/{org}/metadata/delete", webHandler.DeleteMetadata).Methods("DELETE")

	// API prefix
	api := router.PathPrefix("/v1").Subrouter()

	// Public routes (no auth required)
	api.HandleFunc("/orgs", handler.CreateOrganization).Methods("POST")

	// Authenticated API routes (require org token)
	authAPI := api.PathPrefix("").Subrouter()
	authAPI.Use(AuthMiddleware(svc))

	// Organization routes (authenticated)
	authAPI.HandleFunc("/orgs", handler.ListOrganizations).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}", handler.GetOrganization).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}", handler.UpdateOrganization).Methods("PATCH")
	authAPI.HandleFunc("/orgs/{org}", handler.DeleteOrganization).Methods("DELETE")

	// API Key routes (scoped to org, authenticated)
	authAPI.HandleFunc("/orgs/{org}/api-keys", handler.CreateAPIKey).Methods("POST")
	authAPI.HandleFunc("/orgs/{org}/api-keys", handler.ListAPIKeys).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/api-keys/{key_id}", handler.DeleteAPIKey).Methods("DELETE")

	// Project routes (scoped to org, authenticated)
	authAPI.HandleFunc("/orgs/{org}/projects", handler.CreateProject).Methods("POST")
	authAPI.HandleFunc("/orgs/{org}/projects", handler.ListProjects).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}", handler.GetProject).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}", handler.UpdateProject).Methods("PATCH")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}", handler.DeleteProject).Methods("DELETE")

	// Instance routes (scoped to org/project, authenticated)
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/instances", handler.CreateInstance).Methods("POST")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/instances", handler.ListInstances).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/instances/{id}", handler.GetInstance).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/instances/{id}", handler.UpdateInstance).Methods("PATCH")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/instances/{id}", handler.DeleteInstance).Methods("DELETE")

	// Bucket routes (scoped to org/project, authenticated)
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets", handler.CreateBucket).Methods("POST")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets", handler.ListBuckets).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets/{bucket}", handler.GetBucket).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets/{bucket}", handler.UpdateBucket).Methods("PATCH")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets/{bucket}", handler.DeleteBucket).Methods("DELETE")

	// Object routes (scoped to bucket, authenticated)
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets/{bucket}/objects", handler.CreateObject).Methods("POST")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets/{bucket}/objects", handler.ListObjects).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets/{bucket}/objects/{id}", handler.GetObject).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets/{bucket}/objects/{id}", handler.UpdateObject).Methods("PATCH")
	authAPI.HandleFunc("/orgs/{org}/projects/{project}/buckets/{bucket}/objects/{id}", handler.DeleteObject).Methods("DELETE")

	// Metadata routes (scoped to org, authenticated)
	authAPI.HandleFunc("/orgs/{org}/metadata", handler.CreateMetadata).Methods("POST")
	authAPI.HandleFunc("/orgs/{org}/metadata", handler.ListMetadata).Methods("GET").Queries("prefix", "")
	authAPI.HandleFunc("/orgs/{org}/metadata", handler.ListMetadata).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/metadata/{id}", handler.GetMetadata).Methods("GET")
	authAPI.HandleFunc("/orgs/{org}/metadata/{id}", handler.UpdateMetadata).Methods("PATCH")
	authAPI.HandleFunc("/orgs/{org}/metadata/{id}", handler.DeleteMetadata).Methods("DELETE")

	// Terraform state routes (public for now - used by Terraform HTTP backend)
	api.HandleFunc("/tfstate/{id}", handler.TFStateGet).Methods("GET")
	api.HandleFunc("/tfstate/{id}", handler.TFStatePost).Methods("POST")
	api.HandleFunc("/tfstate/{id}", handler.TFStateDelete).Methods("DELETE")
	api.HandleFunc("/tfstate/{id}", handler.TFStateLock).Methods("LOCK")
	api.HandleFunc("/tfstate/{id}", handler.TFStateUnlock).Methods("UNLOCK")

	// Add CORS middleware for development
	router.Use(corsMiddleware)

	// Add logging middleware
	router.Use(loggingMiddleware)

	return router
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
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
