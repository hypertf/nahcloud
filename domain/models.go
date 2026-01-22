package domain

import (
	"time"
)

// Organization represents a top-level organization in the NahCloud system
type Organization struct {
	ID        string    `json:"id" db:"id"`
	Slug      string    `json:"slug" db:"slug"` // URL-safe identifier (like GCP project ID)
	Name      string    `json:"name" db:"name"` // Human-readable display name
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// APIKey represents an API key tied to an organization
type APIKey struct {
	ID         string     `json:"id" db:"id"`
	OrgID      string     `json:"org_id" db:"org_id"`
	Name       string     `json:"name" db:"name"`
	TokenHash  string     `json:"-" db:"token_hash"` // SHA-256 hash, never exposed
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// APIKeyWithToken is returned only on key creation (contains plaintext token)
type APIKeyWithToken struct {
	APIKey
	Token string `json:"token"` // Plaintext token, shown only once
}

// OrganizationWithAPIKey is returned on org creation (includes the initial API key)
type OrganizationWithAPIKey struct {
	Organization
	APIKey APIKeyWithToken `json:"api_key"`
}

// CreateAPIKeyRequest represents the request to create an API key
type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

// Project represents a project in the NahCloud system
type Project struct {
	ID        string    `json:"id" db:"id"`
	OrgID     string    `json:"org_id" db:"org_id"`
	Slug      string    `json:"slug" db:"slug"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Instance represents a compute instance within a project
type Instance struct {
	ID        string    `json:"id" db:"id"`
	ProjectID string    `json:"project_id" db:"project_id"`
	Name      string    `json:"name" db:"name"`
	Region    string    `json:"region" db:"region"`
	CPU       int       `json:"cpu" db:"cpu"`
	MemoryMB  int       `json:"memory_mb" db:"memory_mb"`
	Image     string    `json:"image" db:"image"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// InstanceStatus constants
const (
	StatusRunning = "running"
	StatusStopped = "stopped"
)

// Region constants
const (
	RegionUSEast1    = "us-east-1"
	RegionUSWest1    = "us-west-1"
	RegionEUWest1    = "eu-west-1"
	RegionEUCentral1 = "eu-central-1"
	RegionAPEast1    = "ap-east-1"
)

// ValidRegions is the list of allowed regions
var ValidRegions = []string{
	RegionUSEast1,
	RegionUSWest1,
	RegionEUWest1,
	RegionEUCentral1,
	RegionAPEast1,
}

// Metadata represents key-value metadata storage (org-scoped)
type Metadata struct {
	ID        string    `json:"id" db:"id"`
	OrgID     string    `json:"org_id" db:"org_id"`
	Path      string    `json:"path" db:"path"`
	Value     string    `json:"value" db:"value"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Bucket represents a storage bucket (project-scoped)
// Buckets are logical containers for objects
// Name must be unique within a project
// Objects reference buckets by ID
type Bucket struct {
	ID        string    `json:"id" db:"id"`
	ProjectID string    `json:"project_id" db:"project_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Object represents a stored object within a bucket
// Content is a base64-encoded string
type Object struct {
	ID        string    `json:"id" db:"id"`
	BucketID  string    `json:"bucket_id" db:"bucket_id"`
	Path      string    `json:"path" db:"path"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TFStateLock represents Terraform's HTTP backend lock payload
// Keys are capitalized to match Terraform's expected JSON schema
// See: https://developer.hashicorp.com/terraform/language/state/locking#http-endpoints
type TFStateLock struct {
	ID        string    `json:"ID"`
	Operation string    `json:"Operation,omitempty"`
	Info      string    `json:"Info,omitempty"`
	Who       string    `json:"Who,omitempty"`
	Version   string    `json:"Version,omitempty"`
	Created   time.Time `json:"Created,omitempty"`
	Path      string    `json:"Path,omitempty"`
}

// Organization request/response types

// CreateOrganizationRequest represents the request to create an organization
type CreateOrganizationRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// UpdateOrganizationRequest represents the request to update an organization
type UpdateOrganizationRequest struct {
	Name *string `json:"name,omitempty"`
}

// OrganizationListOptions represents query options for listing organizations
type OrganizationListOptions struct {
	Slug string
}

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name *string `json:"name,omitempty"`
}

// CreateInstanceRequest represents the request to create an instance
type CreateInstanceRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Region    string `json:"region"`
	CPU       int    `json:"cpu"`
	MemoryMB  int    `json:"memory_mb"`
	Image     string `json:"image"`
	Status    string `json:"status,omitempty"`
}

// UpdateInstanceRequest represents the request to update an instance
type UpdateInstanceRequest struct {
	Name     *string `json:"name,omitempty"`
	CPU      *int    `json:"cpu,omitempty"`
	MemoryMB *int    `json:"memory_mb,omitempty"`
	Image    *string `json:"image,omitempty"`
	Status   *string `json:"status,omitempty"`
}

// ProjectListOptions represents query options for listing projects
type ProjectListOptions struct {
	OrgID string
	Slug  string
	Name  string
}

// InstanceListOptions represents query options for listing instances
type InstanceListOptions struct {
	ProjectID string
	Name      string
	Region    string
	Status    string
}

// CreateMetadataRequest represents the request to create metadata
type CreateMetadataRequest struct {
	OrgID string `json:"org_id"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// UpdateMetadataRequest represents the request to update metadata
type UpdateMetadataRequest struct {
	Path  *string `json:"path,omitempty"`
	Value *string `json:"value,omitempty"`
}

// MetadataListOptions represents query options for listing metadata
type MetadataListOptions struct {
	OrgID  string
	Prefix string
}

// CreateBucketRequest represents the request to create a bucket
type CreateBucketRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

// UpdateBucketRequest represents the request to update a bucket
type UpdateBucketRequest struct {
	Name string `json:"name"`
}

// BucketListOptions represents query options for listing buckets
type BucketListOptions struct {
	ProjectID string
	Name      string
}

// CreateObjectRequest represents the request to create an object
type CreateObjectRequest struct {
	BucketID string `json:"bucket_id"`
	Path     string `json:"path"`
	Content  string `json:"content"`
}

// UpdateObjectRequest represents the request to update an object
type UpdateObjectRequest struct {
	Path    *string `json:"path,omitempty"`
	Content *string `json:"content,omitempty"`
}

// ObjectListOptions represents query options for listing objects
type ObjectListOptions struct {
	BucketID string
	Prefix   string
}
