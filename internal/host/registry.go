package host

import (
	"sync"

	"github.com/takaaki-s/claude-code-valet/internal/config"
	"github.com/takaaki-s/claude-code-valet/internal/notify"
	"github.com/takaaki-s/claude-code-valet/internal/session"
)

// SlaveClient is the interface for communicating with remote slave daemons
type SlaveClient interface {
	IsRunning() bool
	ListWithHostID() ([]session.Info, error)
	NotificationHistoryWithHostID() ([]notify.Entry, error)
}

// Host represents a host and its slave client pair
type Host struct {
	ID     string            // "local", "ec2", "docker-dev", etc.
	Type   string            // "local", "ssh", "docker"
	Config config.HostConfig // Host configuration
	Client SlaveClient       // Connection to slave daemon (interface)
}

// Registry manages the list of hosts
type Registry struct {
	mu    sync.RWMutex
	hosts map[string]*Host
	local *Host
}

// NewRegistry builds a HostRegistry from configuration
func NewRegistry(hostConfigs []config.HostConfig) *Registry {
	r := &Registry{
		hosts: make(map[string]*Host),
	}

	// Local host is always registered
	r.local = &Host{
		ID:     "local",
		Type:   "local",
		Config: config.HostConfig{ID: "local", Type: "local"},
	}
	r.hosts["local"] = r.local

	// Register hosts from config (Client is set after tunnel establishment)
	for _, hc := range hostConfigs {
		r.hosts[hc.ID] = &Host{
			ID:     hc.ID,
			Type:   hc.Type,
			Config: hc,
			Client: nil,
		}
	}

	return r
}

// Get retrieves a Host by host ID
func (r *Registry) Get(hostID string) (*Host, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if hostID == "" || hostID == "local" {
		return r.local, true
	}
	h, ok := r.hosts[hostID]
	return h, ok
}

// Local returns the local host
func (r *Registry) Local() *Host {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.local
}

// All returns all hosts (local first)
func (r *Registry) All() []*Host {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := []*Host{r.local}
	for _, h := range r.hosts {
		if h.ID != "local" {
			result = append(result, h)
		}
	}
	return result
}

// Remotes returns only remote hosts
func (r *Registry) Remotes() []*Host {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Host
	for _, h := range r.hosts {
		if h.ID != "local" {
			result = append(result, h)
		}
	}
	return result
}

// AllIDs returns all host IDs
func (r *Registry) AllIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := []string{"local"}
	for id := range r.hosts {
		if id != "local" {
			ids = append(ids, id)
		}
	}
	return ids
}

// SetClient sets the slave client for a host (called after tunnel establishment)
func (r *Registry) SetClient(hostID string, client SlaveClient) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if h, ok := r.hosts[hostID]; ok {
		h.Client = client
	}
}

// IsConnected returns whether the host is connected
func (r *Registry) IsConnected(hostID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	h, ok := r.hosts[hostID]
	if !ok {
		return false
	}
	return h.Client != nil && h.Client.IsRunning()
}
