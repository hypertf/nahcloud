package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"

	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/pkg/endec"
)

// Service provides business logic for NahCloud operations
type Service struct {
	orgRepo      OrganizationRepository
	apiKeyRepo   APIKeyRepository
	projectRepo  ProjectRepository
	instanceRepo InstanceRepository
	metadataRepo MetadataRepository
	bucketRepo   BucketRepository
	objectRepo   ObjectRepository
}

// OrganizationRepository defines the interface for organization data operations
type OrganizationRepository interface {
	Create(org *domain.Organization) error
	GetByID(id string) (*domain.Organization, error)
	GetBySlug(slug string) (*domain.Organization, error)
	List(opts domain.OrganizationListOptions) ([]*domain.Organization, error)
	Update(id string, req domain.UpdateOrganizationRequest) (*domain.Organization, error)
	Delete(id string) error
}

// APIKeyRepository defines the interface for API key data operations
type APIKeyRepository interface {
	Create(key *domain.APIKey) error
	GetByID(id string) (*domain.APIKey, error)
	GetByTokenHash(tokenHash string) (*domain.APIKey, error)
	ListByOrgID(orgID string) ([]*domain.APIKey, error)
	UpdateLastUsed(id string) error
	Delete(id string) error
}

// ProjectRepository defines the interface for project data operations
type ProjectRepository interface {
	Create(project *domain.Project) error
	GetByID(id string) (*domain.Project, error)
	GetBySlug(orgID, slug string) (*domain.Project, error)
	GetByName(name string) (*domain.Project, error)
	List(opts domain.ProjectListOptions) ([]*domain.Project, error)
	Update(id string, req domain.UpdateProjectRequest) (*domain.Project, error)
	Delete(id string) error
}

// InstanceRepository defines the interface for instance data operations
type InstanceRepository interface {
	Create(instance *domain.Instance) error
	GetByID(id string) (*domain.Instance, error)
	List(opts domain.InstanceListOptions) ([]*domain.Instance, error)
	Update(id string, req domain.UpdateInstanceRequest) (*domain.Instance, error)
	Delete(id string) error
}

// MetadataRepository defines the interface for metadata data operations
type MetadataRepository interface {
	Create(req domain.CreateMetadataRequest) (*domain.Metadata, error)
	GetByID(id string) (*domain.Metadata, error)
	GetByPath(orgID, path string) (*domain.Metadata, error)
	Update(id string, req domain.UpdateMetadataRequest) (*domain.Metadata, error)
	List(opts domain.MetadataListOptions) ([]*domain.Metadata, error)
	Delete(id string) error
}

// BucketRepository defines the interface for bucket data operations
type BucketRepository interface {
	Create(bucket *domain.Bucket) error
	GetByID(id string) (*domain.Bucket, error)
	GetByName(projectID, name string) (*domain.Bucket, error)
	List(opts domain.BucketListOptions) ([]*domain.Bucket, error)
	Update(id string, req domain.UpdateBucketRequest) (*domain.Bucket, error)
	Delete(id string) error
}

// ObjectRepository defines the interface for object data operations
type ObjectRepository interface {
	Create(req domain.CreateObjectRequest) (*domain.Object, error)
	GetByID(id string) (*domain.Object, error)
	Update(id string, req domain.UpdateObjectRequest) (*domain.Object, error)
	List(opts domain.ObjectListOptions) ([]*domain.Object, error)
	Delete(id string) error
}

// NewService creates a new service instance
func NewService(orgRepo OrganizationRepository, apiKeyRepo APIKeyRepository, projectRepo ProjectRepository, instanceRepo InstanceRepository, metadataRepo MetadataRepository, bucketRepo BucketRepository, objectRepo ObjectRepository) *Service {
	return &Service{
		orgRepo:      orgRepo,
		apiKeyRepo:   apiKeyRepo,
		projectRepo:  projectRepo,
		instanceRepo: instanceRepo,
		metadataRepo: metadataRepo,
		bucketRepo:   bucketRepo,
		objectRepo:   objectRepo,
	}
}

// generateID generates a random hex ID
func generateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// validateSlug validates a slug (used for org and project slugs)
func validateSlug(slug string) error {
	if slug == "" {
		return domain.InvalidInputError("slug cannot be empty", nil)
	}
	if len(slug) > 63 {
		return domain.InvalidInputError("slug too long", map[string]interface{}{
			"max_length": 63,
			"actual":     len(slug),
		})
	}
	// Slug must be lowercase alphanumeric with dashes, starting with a letter
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(slug) {
		return domain.InvalidInputError("slug must start with a lowercase letter and contain only lowercase letters, numbers, and dashes", nil)
	}
	return nil
}

// validateName validates a resource name
func validateName(name string, resourceType string) error {
	if name == "" {
		return domain.InvalidInputError(resourceType+" name cannot be empty", nil)
	}
	if len(name) > 255 {
		return domain.InvalidInputError(resourceType+" name too long", map[string]interface{}{
			"max_length": 255,
			"actual":     len(name),
		})
	}
	return nil
}

// validateProjectName validates a project name
func validateProjectName(name string) error {
	if name == "" {
		return domain.InvalidInputError("project name cannot be empty", nil)
	}
	if len(name) > 255 {
		return domain.InvalidInputError("project name too long", map[string]interface{}{
			"max_length": 255,
			"actual":     len(name),
		})
	}
	// Simple alphanumeric + dash/underscore validation
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(name) {
		return domain.InvalidInputError("project name can only contain alphanumeric characters, dashes, and underscores", nil)
	}
	return nil
}

// validateBucketName validates a bucket name
func validateBucketName(name string) error {
	if name == "" {
		return domain.InvalidInputError("bucket name cannot be empty", nil)
	}
	if len(name) > 255 {
		return domain.InvalidInputError("bucket name too long", map[string]interface{}{
			"max_length": 255,
			"actual":     len(name),
		})
	}
	// Simple alphanumeric + dash/underscore validation
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(name) {
		return domain.InvalidInputError("bucket name can only contain alphanumeric characters, dashes, and underscores", nil)
	}
	return nil
}

// validateInstanceName validates an instance name
func validateInstanceName(name string) error {
	if name == "" {
		return domain.InvalidInputError("instance name cannot be empty", nil)
	}
	if len(name) > 255 {
		return domain.InvalidInputError("instance name too long", map[string]interface{}{
			"max_length": 255,
			"actual":     len(name),
		})
	}
	// Simple alphanumeric + dash/underscore validation
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(name) {
		return domain.InvalidInputError("instance name can only contain alphanumeric characters, dashes, and underscores", nil)
	}
	return nil
}

// validateInstanceSpecs validates instance specifications
func validateInstanceSpecs(cpu int, memoryMB int, image string) error {
	if cpu <= 0 {
		return domain.InvalidInputError("CPU must be positive", map[string]interface{}{"cpu": cpu})
	}
	if cpu > 64 {
		return domain.InvalidInputError("CPU too high", map[string]interface{}{
			"max_cpu": 64,
			"actual":  cpu,
		})
	}
	if memoryMB <= 0 {
		return domain.InvalidInputError("memory must be positive", map[string]interface{}{"memory_mb": memoryMB})
	}
	if memoryMB > 512*1024 { // 512GB
		return domain.InvalidInputError("memory too high", map[string]interface{}{
			"max_memory_mb": 512 * 1024,
			"actual":        memoryMB,
		})
	}
	if image == "" {
		return domain.InvalidInputError("image cannot be empty", nil)
	}
	if len(image) > 255 {
		return domain.InvalidInputError("image name too long", map[string]interface{}{
			"max_length": 255,
			"actual":     len(image),
		})
	}
	return nil
}

// validateInstanceStatus validates instance status
func validateInstanceStatus(status string) error {
	if status != domain.StatusRunning && status != domain.StatusStopped {
		return domain.InvalidInputError("invalid status", map[string]interface{}{
			"valid_statuses": []string{domain.StatusRunning, domain.StatusStopped},
			"actual":         status,
		})
	}
	return nil
}

// validateInstanceRegion validates instance region
func validateInstanceRegion(region string) error {
	if region == "" {
		return domain.InvalidInputError("region is required", nil)
	}
	for _, validRegion := range domain.ValidRegions {
		if region == validRegion {
			return nil
		}
	}
	return domain.InvalidInputError("invalid region", map[string]interface{}{
		"valid_regions": domain.ValidRegions,
		"actual":        region,
	})
}

// validateObjectPath validates an object path
func validateObjectPath(path string) error {
	if path == "" {
		return domain.InvalidInputError("object path cannot be empty", nil)
	}
	if len(path) > 1024 {
		return domain.InvalidInputError("object path too long", map[string]interface{}{"max_length": 1024, "actual": len(path)})
	}
	return nil
}

// Organization operations

// hashToken returns the SHA-256 hash of a token as a hex string
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// CreateOrganization creates a new organization with an initial API key
func (s *Service) CreateOrganization(req domain.CreateOrganizationRequest) (*domain.OrganizationWithAPIKey, error) {
	if err := validateSlug(req.Slug); err != nil {
		return nil, err
	}
	if err := validateName(req.Name, "organization"); err != nil {
		return nil, err
	}

	orgID, err := generateID()
	if err != nil {
		return nil, domain.InternalError("failed to generate org ID")
	}

	org := &domain.Organization{
		ID:   orgID,
		Slug: req.Slug,
		Name: req.Name,
	}

	if err := s.orgRepo.Create(org); err != nil {
		return nil, err
	}

	// Create initial API key for the org
	apiKeyWithToken, err := s.createAPIKey(orgID, "default")
	if err != nil {
		// Rollback org creation on API key failure
		s.orgRepo.Delete(orgID)
		return nil, err
	}

	return &domain.OrganizationWithAPIKey{
		Organization: *org,
		APIKey:       *apiKeyWithToken,
	}, nil
}

// GetOrganization retrieves an organization by ID
func (s *Service) GetOrganization(id string) (*domain.Organization, error) {
	return s.orgRepo.GetByID(id)
}

// GetOrganizationBySlug retrieves an organization by slug
func (s *Service) GetOrganizationBySlug(slug string) (*domain.Organization, error) {
	return s.orgRepo.GetBySlug(slug)
}

// GetOrganizationByToken retrieves an organization by validating the provided API key token
func (s *Service) GetOrganizationByToken(token string) (*domain.Organization, error) {
	// Validate token format (must be an API key)
	if _, err := endec.ValidateToken(token, endec.PrefixAPI); err != nil {
		return nil, domain.UnauthorizedError("invalid token format")
	}

	apiKey, err := s.apiKeyRepo.GetByTokenHash(hashToken(token))
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.UnauthorizedError("invalid token")
		}
		return nil, err
	}

	// Update last used timestamp (fire and forget)
	go s.apiKeyRepo.UpdateLastUsed(apiKey.ID)

	return s.orgRepo.GetByID(apiKey.OrgID)
}

// ListOrganizations lists organizations with optional filtering
func (s *Service) ListOrganizations(opts domain.OrganizationListOptions) ([]*domain.Organization, error) {
	return s.orgRepo.List(opts)
}

// UpdateOrganization updates an existing organization
func (s *Service) UpdateOrganization(id string, req domain.UpdateOrganizationRequest) (*domain.Organization, error) {
	if req.Name != nil {
		if err := validateName(*req.Name, "organization"); err != nil {
			return nil, err
		}
	}

	return s.orgRepo.Update(id, req)
}

// DeleteOrganization deletes an organization
func (s *Service) DeleteOrganization(id string) error {
	return s.orgRepo.Delete(id)
}

// API Key operations

// createAPIKey is an internal helper to create an API key
func (s *Service) createAPIKey(orgID, name string) (*domain.APIKeyWithToken, error) {
	keyID, err := generateID()
	if err != nil {
		return nil, domain.InternalError("failed to generate API key ID")
	}

	// Generate API token (24 bytes = 192 bits of entropy)
	token, err := endec.CreateToken(endec.PrefixAPI, 24)
	if err != nil {
		return nil, domain.InternalError("failed to generate token")
	}

	apiKey := &domain.APIKey{
		ID:        keyID,
		OrgID:     orgID,
		Name:      name,
		TokenHash: hashToken(token),
	}

	if err := s.apiKeyRepo.Create(apiKey); err != nil {
		return nil, err
	}

	return &domain.APIKeyWithToken{
		APIKey: *apiKey,
		Token:  token,
	}, nil
}

// CreateAPIKey creates a new API key for an organization
func (s *Service) CreateAPIKey(orgID string, req domain.CreateAPIKeyRequest) (*domain.APIKeyWithToken, error) {
	// Verify organization exists
	if _, err := s.orgRepo.GetByID(orgID); err != nil {
		return nil, err
	}

	name := req.Name
	if name == "" {
		name = "unnamed"
	}

	return s.createAPIKey(orgID, name)
}

// ListAPIKeys lists all API keys for an organization
func (s *Service) ListAPIKeys(orgID string) ([]*domain.APIKey, error) {
	return s.apiKeyRepo.ListByOrgID(orgID)
}

// DeleteAPIKey deletes an API key
func (s *Service) DeleteAPIKey(orgID, keyID string) error {
	// Verify the key belongs to the org
	key, err := s.apiKeyRepo.GetByID(keyID)
	if err != nil {
		return err
	}
	if key.OrgID != orgID {
		return domain.NotFoundError("api_key", keyID)
	}

	return s.apiKeyRepo.Delete(keyID)
}

// Project operations

// CreateProject creates a new project within an organization
func (s *Service) CreateProject(orgID string, req domain.CreateProjectRequest) (*domain.Project, error) {
	if err := validateSlug(req.Slug); err != nil {
		return nil, err
	}
	if err := validateName(req.Name, "project"); err != nil {
		return nil, err
	}

	// Verify organization exists
	_, err := s.orgRepo.GetByID(orgID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ForeignKeyViolationError("organization", "id", orgID)
		}
		return nil, err
	}

	id, err := generateID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}

	project := &domain.Project{
		ID:    id,
		OrgID: orgID,
		Slug:  req.Slug,
		Name:  req.Name,
	}

	if err := s.projectRepo.Create(project); err != nil {
		return nil, err
	}

	return project, nil
}

// GetProject retrieves a project by ID
func (s *Service) GetProject(id string) (*domain.Project, error) {
	return s.projectRepo.GetByID(id)
}

// GetProjectBySlug retrieves a project by org ID and slug
func (s *Service) GetProjectBySlug(orgID, slug string) (*domain.Project, error) {
	return s.projectRepo.GetBySlug(orgID, slug)
}

// ListProjects lists projects with optional filtering
func (s *Service) ListProjects(opts domain.ProjectListOptions) ([]*domain.Project, error) {
	return s.projectRepo.List(opts)
}

// UpdateProject updates an existing project
func (s *Service) UpdateProject(id string, req domain.UpdateProjectRequest) (*domain.Project, error) {
	if req.Name != nil {
		if err := validateName(*req.Name, "project"); err != nil {
			return nil, err
		}
	}

	return s.projectRepo.Update(id, req)
}

// DeleteProject deletes a project
func (s *Service) DeleteProject(id string) error {
	return s.projectRepo.Delete(id)
}

// Instance operations

// CreateInstance creates a new instance
func (s *Service) CreateInstance(req domain.CreateInstanceRequest) (*domain.Instance, error) {
	if err := validateInstanceName(req.Name); err != nil {
		return nil, err
	}

	if err := validateInstanceRegion(req.Region); err != nil {
		return nil, err
	}

	if err := validateInstanceSpecs(req.CPU, req.MemoryMB, req.Image); err != nil {
		return nil, err
	}

	status := req.Status
	if status == "" {
		status = domain.StatusRunning
	}
	if err := validateInstanceStatus(status); err != nil {
		return nil, err
	}

	// Verify project exists
	_, err := s.projectRepo.GetByID(req.ProjectID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ForeignKeyViolationError("project", "id", req.ProjectID)
		}
		return nil, err
	}

	id, err := generateID()
	if err != nil {
		return nil, domain.InternalError("failed to generate ID")
	}

	instance := &domain.Instance{
		ID:        id,
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Region:    req.Region,
		CPU:       req.CPU,
		MemoryMB:  req.MemoryMB,
		Image:     req.Image,
		Status:    status,
	}

	if err := s.instanceRepo.Create(instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// GetInstance retrieves an instance by ID
func (s *Service) GetInstance(id string) (*domain.Instance, error) {
	return s.instanceRepo.GetByID(id)
}

// ListInstances lists instances with optional filtering
func (s *Service) ListInstances(opts domain.InstanceListOptions) ([]*domain.Instance, error) {
	return s.instanceRepo.List(opts)
}

// UpdateInstance updates an existing instance
func (s *Service) UpdateInstance(id string, req domain.UpdateInstanceRequest) (*domain.Instance, error) {
	// Get current instance to check for immutable field changes
	current, err := s.instanceRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Image changes require instance recreation - only error if actually changing
	if req.Image != nil && *req.Image != current.Image {
		return nil, domain.InvalidInputError(
			"Cannot change instance image from '"+current.Image+"' to '"+*req.Image+"'. The image field is immutable - you must destroy and recreate the instance to change the image.",
			map[string]interface{}{
				"field":           "image",
				"current_value":   current.Image,
				"requested_value": *req.Image,
				"solution":        "Remove and re-add the resource, or use 'terraform taint' to force recreation",
			},
		)
	}

	if req.Name != nil {
		if err := validateInstanceName(*req.Name); err != nil {
			return nil, err
		}
	}

	if req.CPU != nil || req.MemoryMB != nil {
		// Validate complete specs using current values as defaults
		cpu := current.CPU
		memory := current.MemoryMB
		image := current.Image

		if req.CPU != nil {
			cpu = *req.CPU
		}
		if req.MemoryMB != nil {
			memory = *req.MemoryMB
		}

		if err := validateInstanceSpecs(cpu, memory, image); err != nil {
			return nil, err
		}
	}

	if req.Status != nil {
		if err := validateInstanceStatus(*req.Status); err != nil {
			return nil, err
		}
	}

	return s.instanceRepo.Update(id, req)
}

// DeleteInstance deletes an instance
func (s *Service) DeleteInstance(id string) error {
	return s.instanceRepo.Delete(id)
}

// Metadata operations

// CreateMetadata creates new metadata
func (s *Service) CreateMetadata(req domain.CreateMetadataRequest) (*domain.Metadata, error) {
	if req.OrgID == "" {
		return nil, domain.InvalidInputError("org_id cannot be empty", nil)
	}
	if req.Path == "" {
		return nil, domain.InvalidInputError("metadata path cannot be empty", nil)
	}

	// Verify organization exists
	_, err := s.orgRepo.GetByID(req.OrgID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ForeignKeyViolationError("organization", "id", req.OrgID)
		}
		return nil, err
	}

	return s.metadataRepo.Create(req)
}

// GetMetadata retrieves metadata by ID
func (s *Service) GetMetadata(id string) (*domain.Metadata, error) {
	if id == "" {
		return nil, domain.InvalidInputError("metadata ID cannot be empty", nil)
	}

	return s.metadataRepo.GetByID(id)
}

// GetMetadataByPath retrieves metadata by org ID and path
func (s *Service) GetMetadataByPath(orgID, path string) (*domain.Metadata, error) {
	if orgID == "" {
		return nil, domain.InvalidInputError("org_id cannot be empty", nil)
	}
	if path == "" {
		return nil, domain.InvalidInputError("metadata path cannot be empty", nil)
	}

	return s.metadataRepo.GetByPath(orgID, path)
}

// UpdateMetadata updates existing metadata
func (s *Service) UpdateMetadata(id string, req domain.UpdateMetadataRequest) (*domain.Metadata, error) {
	if id == "" {
		return nil, domain.InvalidInputError("metadata ID cannot be empty", nil)
	}

	return s.metadataRepo.Update(id, req)
}

// ListMetadata lists metadata with optional prefix filtering
func (s *Service) ListMetadata(opts domain.MetadataListOptions) ([]*domain.Metadata, error) {
	return s.metadataRepo.List(opts)
}

// DeleteMetadata deletes metadata by ID
func (s *Service) DeleteMetadata(id string) error {
	if id == "" {
		return domain.InvalidInputError("metadata ID cannot be empty", nil)
	}

	return s.metadataRepo.Delete(id)
}

// Bucket operations

// CreateBucket creates a new bucket within a project
func (s *Service) CreateBucket(projectID string, req domain.CreateBucketRequest) (*domain.Bucket, error) {
	if err := validateBucketName(req.Name); err != nil {
		return nil, err
	}

	// Verify project exists
	_, err := s.projectRepo.GetByID(projectID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ForeignKeyViolationError("project", "id", projectID)
		}
		return nil, err
	}

	// Use name as the stable identifier (ID) - scoped by project
	b := &domain.Bucket{ID: req.Name, ProjectID: projectID, Name: req.Name}
	if err := s.bucketRepo.Create(b); err != nil {
		return nil, err
	}
	return b, nil
}

// GetBucket retrieves a bucket by ID (or name, if name is the identifier)
func (s *Service) GetBucket(id string) (*domain.Bucket, error) {
	return s.bucketRepo.GetByID(id)
}

// GetBucketByName retrieves a bucket by project ID and name
func (s *Service) GetBucketByName(projectID, name string) (*domain.Bucket, error) {
	return s.bucketRepo.GetByName(projectID, name)
}

// ListBuckets lists buckets with optional filtering
func (s *Service) ListBuckets(opts domain.BucketListOptions) ([]*domain.Bucket, error) {
	return s.bucketRepo.List(opts)
}

// UpdateBucket updates an existing bucket
// With IDs equal to names, bucket name is immutable. Attempting to change it will return an error.
func (s *Service) UpdateBucket(id string, req domain.UpdateBucketRequest) (*domain.Bucket, error) {
	if err := validateBucketName(req.Name); err != nil {
		return nil, err
	}
	// Get current bucket to enforce immutability
	current, err := s.bucketRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if req.Name != current.Name {
		return nil, domain.InvalidInputError(
			"Cannot change bucket name from '"+current.Name+"' to '"+req.Name+"'. The name is immutable because it is used as the bucket ID. Destroy and recreate the bucket to change the name.",
			map[string]interface{}{
				"field":           "name",
				"current_value":   current.Name,
				"requested_value": req.Name,
				"solution":        "Remove and re-add the resource, or recreate the bucket with the desired name",
			},
		)
	}
	// No-op update (name unchanged)
	return current, nil
}

// DeleteBucket deletes a bucket
func (s *Service) DeleteBucket(id string) error {
	return s.bucketRepo.Delete(id)
}

// Object operations

// CreateObject creates a new object under a bucket
func (s *Service) CreateObject(req domain.CreateObjectRequest) (*domain.Object, error) {
	if err := validateObjectPath(req.Path); err != nil {
		return nil, err
	}
	if req.BucketID == "" {
		return nil, domain.InvalidInputError("bucket_id cannot be empty", nil)
	}
	if req.Content == "" {
		return nil, domain.InvalidInputError("content cannot be empty", nil)
	}
	// Verify bucket exists
	if _, err := s.bucketRepo.GetByID(req.BucketID); err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ForeignKeyViolationError("bucket", "id", req.BucketID)
		}
		return nil, err
	}
	return s.objectRepo.Create(req)
}

// GetObject retrieves an object by ID
func (s *Service) GetObject(id string) (*domain.Object, error) {
	return s.objectRepo.GetByID(id)
}

// ListObjects lists objects with optional filtering
func (s *Service) ListObjects(opts domain.ObjectListOptions) ([]*domain.Object, error) {
	return s.objectRepo.List(opts)
}

// UpdateObject updates an existing object
func (s *Service) UpdateObject(id string, req domain.UpdateObjectRequest) (*domain.Object, error) {
	if req.Path != nil {
		if err := validateObjectPath(*req.Path); err != nil {
			return nil, err
		}
	}
	return s.objectRepo.Update(id, req)
}

// DeleteObject deletes an object
func (s *Service) DeleteObject(id string) error {
	return s.objectRepo.Delete(id)
}
