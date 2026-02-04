package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Target struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	DB       string `json:"db"`
	Encoding string `json:"encoding"`
}

type Config struct {
	Targets []Target `json:"targets"`
}

type ConfigManager struct {
	path string
	mu   sync.RWMutex
	data Config
}

func NewConfigManager() (*ConfigManager, error) {
	// Store config in user's config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	
	appDir := filepath.Join(configDir, "OpenZ3950Desktop")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(appDir, "config.json")
	cm := &ConfigManager{path: path}
	
	// Load existing or set defaults
	if err := cm.Load(); err != nil {
		// Default targets
		cm.data.Targets = []Target{
			{Name: "Library of Congress", Host: "lx2.loc.gov", Port: 210, DB: "LCDB", Encoding: "MARC21"},
			{Name: "Oxford University", Host: "library.ox.ac.uk", Port: 210, DB: "MAIN_BIB", Encoding: "MARC21"},
		}
		cm.Save()
	}
	
	return cm, nil
}

func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	data, err := os.ReadFile(cm.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &cm.data)
}

func (cm *ConfigManager) Save() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	data, err := json.MarshalIndent(cm.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cm.path, data, 0644)
}

func (cm *ConfigManager) AddTarget(t Target) error {
	cm.mu.Lock()
	cm.data.Targets = append(cm.data.Targets, t)
	cm.mu.Unlock()
	return cm.Save()
}

func (cm *ConfigManager) GetTarget(name string) (Target, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for _, t := range cm.data.Targets {
		if t.Name == name {
			return t, true
		}
	}
	return Target{}, false
}

func (cm *ConfigManager) ListTargets() []Target {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	// Return copy
	list := make([]Target, len(cm.data.Targets))
	copy(list, cm.data.Targets)
	return list
}

func (cm *ConfigManager) DeleteTarget(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	newTargets := make([]Target, 0)
	for _, t := range cm.data.Targets {
		if t.Name != name {
			newTargets = append(newTargets, t)
		}
	}
	
	cm.data.Targets = newTargets
	return cm.Save()
}
