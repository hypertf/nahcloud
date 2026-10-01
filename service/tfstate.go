package service

import (
	"encoding/base64"
	"encoding/json"

	"github.com/hypertf/nahcloud/domain"
)

func tfStatePath(stateID string) string {
	return ".nahcloud/tfstate/" + base64.RawURLEncoding.EncodeToString([]byte(stateID)) + "/state"
}

func tfStateLockPath(stateID string) string {
	return ".nahcloud/tfstate/" + base64.RawURLEncoding.EncodeToString([]byte(stateID)) + "/lock"
}

// metadataByExactPath finds metadata by exact path using prefix listing
func (s *Service) metadataByExactPath(orgID, path string) (*domain.Metadata, error) {
	return s.metadataRepo.GetByPath(orgID, path)
}

// GetTFState returns the raw state JSON for a given state ID
func (s *Service) GetTFState(orgID, stateID string) (string, error) {
	m, err := s.metadataByExactPath(orgID, tfStatePath(stateID))
	if err != nil {
		return "", err
	}
	return m.Value, nil
}

// SetTFState creates or updates the state JSON for a given state ID
func (s *Service) SetTFState(orgID, stateID string, stateJSON string) error {
	path := tfStatePath(stateID)
	m, err := s.metadataByExactPath(orgID, path)
	if err != nil {
		if domain.IsNotFound(err) {
			_, err := s.metadataRepo.Create(domain.CreateMetadataRequest{OrgID: orgID, Path: path, Value: stateJSON})
			return err
		}
		return err
	}
	// Update existing
	return s.updateMetadataValue(m.ID, stateJSON)
}

// DeleteTFState deletes the state entry if it exists
func (s *Service) DeleteTFState(orgID, stateID string) error {
	path := tfStatePath(stateID)
	m, err := s.metadataByExactPath(orgID, path)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil
		}
		return err
	}
	return s.metadataRepo.Delete(m.ID)
}

// GetTFStateLock returns the current lock JSON and parsed lock info if present
func (s *Service) GetTFStateLock(orgID, stateID string) (string, *domain.TFStateLock, error) {
	m, err := s.metadataByExactPath(orgID, tfStateLockPath(stateID))
	if err != nil {
		return "", nil, err
	}
	var li domain.TFStateLock
	if err := json.Unmarshal([]byte(m.Value), &li); err != nil {
		// If stored value isn't valid JSON, still return raw
		return m.Value, nil, nil
	}
	return m.Value, &li, nil
}

// TryLockTFState attempts to acquire a lock; returns existing lock JSON if already locked
func (s *Service) TryLockTFState(orgID, stateID string, lockJSON string) (alreadyLocked bool, existingLockJSON string, err error) {
	path := tfStateLockPath(stateID)
	_, err = s.metadataRepo.Create(domain.CreateMetadataRequest{OrgID: orgID, Path: path, Value: lockJSON})
	if err == nil {
		return false, "", nil
	}
	if !domain.IsAlreadyExists(err) {
		return false, "", err
	}

	m, err := s.metadataByExactPath(orgID, path)
	if err != nil {
		return false, "", err
	}
	return true, m.Value, nil
}

// UnlockTFState removes the lock if present
func (s *Service) UnlockTFState(orgID, stateID string) (existed bool, lockJSON string, err error) {
	path := tfStateLockPath(stateID)
	m, err := s.metadataByExactPath(orgID, path)
	if err != nil {
		if domain.IsNotFound(err) {
			return false, "", nil
		}
		return false, "", err
	}
	if err := s.metadataRepo.Delete(m.ID); err != nil {
		return false, "", err
	}
	return true, m.Value, nil
}

// updateMetadataValue is a tiny helper to update only value by ID
func (s *Service) updateMetadataValue(id string, value string) error {
	req := domain.UpdateMetadataRequest{Value: &value}
	_, err := s.metadataRepo.Update(id, req)
	return err
}
