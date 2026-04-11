package runner

import "strings"

type HostLoad struct {
	Host    string
	Running int
}

type HostSelection struct {
	Hosts      []string
	MaxPerHost *int
}

func (s HostSelection) Select(preferred string, loads []HostLoad) (string, bool) {
	hosts := normalizeHosts(s.Hosts)
	if len(hosts) == 0 {
		return "", true
	}

	counts := hostCounts(loads)
	preferred = strings.TrimSpace(preferred)
	if preferred != "" && containsHost(hosts, preferred) && s.hasCapacity(preferred, counts[preferred]) {
		return preferred, true
	}

	bestHost := ""
	bestLoad := 0
	for _, host := range hosts {
		load := counts[host]
		if !s.hasCapacity(host, load) {
			continue
		}
		if bestHost == "" || load < bestLoad {
			bestHost = host
			bestLoad = load
		}
	}
	if bestHost == "" {
		return "", false
	}
	return bestHost, true
}

func (s HostSelection) hasCapacity(_ string, running int) bool {
	if s.MaxPerHost == nil {
		return true
	}
	return running < *s.MaxPerHost
}

func normalizeHosts(hosts []string) []string {
	normalized := make([]string, 0, len(hosts))
	for _, host := range hosts {
		trimmed := strings.TrimSpace(host)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func hostCounts(loads []HostLoad) map[string]int {
	counts := make(map[string]int, len(loads))
	for _, load := range loads {
		host := strings.TrimSpace(load.Host)
		if host == "" || load.Running <= 0 {
			continue
		}
		counts[host] += load.Running
	}
	return counts
}

func containsHost(hosts []string, host string) bool {
	for _, candidate := range hosts {
		if candidate == host {
			return true
		}
	}
	return false
}
