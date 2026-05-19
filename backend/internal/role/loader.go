package role

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadFromYAML loads a role from a YAML file.
func LoadFromYAML(path string) (*Role, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read role file: %w", err)
	}
	var role Role
	if err := yaml.Unmarshal(data, &role); err != nil {
		return nil, fmt.Errorf("failed to parse role YAML: %w", err)
	}
	if err := role.Validate(); err != nil {
		return nil, err
	}
	return &role, nil
}

// LoadFromDirectory loads all roles from a directory into a pool.
func LoadFromDirectory(pool *Pool, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to list role files: %w", err)
	}
	for _, file := range files {
		role, err := LoadFromYAML(file)
		if err != nil {
			return fmt.Errorf("failed to load role from %s: %w", file, err)
		}
		if err := pool.Add(role); err != nil {
			return err
		}
	}
	return nil
}

// LoadCompanyRoles loads roles for a specific company with fallback chain.
// Fallback order: company-specific -> industry default -> global default
func LoadCompanyRoles(baseDir, companyID, industry string) (*Pool, error) {
	pool := NewPool()

	// Try company-specific config first
	companyRoleDir := filepath.Join(baseDir, "companies", companyID, "roles")
	if _, err := os.Stat(companyRoleDir); err == nil {
		if err := LoadFromDirectory(pool, companyRoleDir); err != nil {
			return nil, fmt.Errorf("failed to load company roles: %w", err)
		}
		return pool, nil
	}

	// Fallback to industry default
	if industry != "" {
		industryDir := filepath.Join(baseDir, "industries", industry, "roles")
		if _, err := os.Stat(industryDir); err == nil {
			if err := LoadFromDirectory(pool, industryDir); err != nil {
				return nil, fmt.Errorf("failed to load industry roles: %w", err)
			}
			return pool, nil
		}
	}

	// Final fallback to default
	defaultDir := filepath.Join(baseDir, "roles")
	if _, err := os.Stat(defaultDir); err == nil {
		if err := LoadFromDirectory(pool, defaultDir); err != nil {
			return nil, fmt.Errorf("failed to load default roles: %w", err)
		}
	}

	return pool, nil
}