package proxy

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func ShortName(displayName string) string {
	s := strings.ToLower(displayName)
	s = nonAlphaNum.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}

type routeEntry struct {
	ServerID string
	ToolName string
}

type RoutingTable struct {
	routes     map[string]routeEntry
	shortNames map[string]int
	mu         sync.RWMutex
}

func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		routes:     make(map[string]routeEntry),
		shortNames: make(map[string]int),
	}
}

func (rt *RoutingTable) AddServer(serverID, displayName string, toolNames []string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	base := ShortName(displayName)
	prefix := base

	rt.shortNames[base]++
	if rt.shortNames[base] > 1 {
		prefix = fmt.Sprintf("%s_%d", base, rt.shortNames[base])
	}

	for _, tool := range toolNames {
		namespacedName := prefix + "__" + tool
		rt.routes[namespacedName] = routeEntry{
			ServerID: serverID,
			ToolName: tool,
		}
	}
}

func (rt *RoutingTable) RemoveServer(serverID string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for k, v := range rt.routes {
		if v.ServerID == serverID {
			delete(rt.routes, k)
		}
	}
}

func (rt *RoutingTable) Resolve(namespacedTool string) (serverID, toolName string, ok bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	entry, ok := rt.routes[namespacedTool]
	if !ok {
		return "", "", false
	}
	return entry.ServerID, entry.ToolName, true
}

func (rt *RoutingTable) AllEntries() map[string]routeEntry {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	result := make(map[string]routeEntry, len(rt.routes))
	for k, v := range rt.routes {
		result[k] = v
	}
	return result
}

func (rt *RoutingTable) AllTools() map[string]string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	result := make(map[string]string, len(rt.routes))
	for k, v := range rt.routes {
		result[k] = v.ServerID
	}
	return result
}
