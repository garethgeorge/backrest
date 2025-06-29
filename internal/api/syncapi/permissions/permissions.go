package permissions

import (
	"fmt"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

type ScopeSet struct {
	plans map[string]struct{}
	repos map[string]struct{}

	excludedPlans map[string]struct{}
	excludedRepos map[string]struct{}

	wildcard bool
}

func NewScopeSet(scopes []string) (*ScopeSet, error) {
	scopeSet := &ScopeSet{
		plans:         make(map[string]struct{}),
		repos:         make(map[string]struct{}),
		excludedPlans: make(map[string]struct{}),
		excludedRepos: make(map[string]struct{}),
		wildcard:      false,
	}

	for _, scope := range scopes {
		if scope == "*" {
			scopeSet.wildcard = true
		} else if len(scope) > 5 && strings.HasPrefix(scope, "repo:") {
			scopeSet.repos[scope[len("repo:"):]] = struct{}{}
		} else if len(scope) > 5 && strings.HasPrefix(scope, "plan:") {
			scopeSet.plans[scope[len("plan:"):]] = struct{}{}
		} else if len(scope) > 6 && strings.HasPrefix(scope, "!repo:") {
			scopeSet.excludedRepos[scope[len("!repo:"):]] = struct{}{}
		} else if len(scope) > 6 && strings.HasPrefix(scope, "!plan:") {
			scopeSet.excludedPlans[scope[len("!plan:"):]] = struct{}{}
		} else {
			return nil, fmt.Errorf("invalid scope format: %s", scope)
		}
	}

	return scopeSet, nil
}

func (s *ScopeSet) ContainsPlan(planID string) bool {
	if _, ok := s.excludedPlans[planID]; ok {
		return false
	}
	if s.wildcard {
		return true
	}
	if _, ok := s.plans[planID]; ok {
		return true
	}
	return false
}

func (s *ScopeSet) ContainsRepo(repoID string) bool {
	if _, ok := s.excludedRepos[repoID]; ok {
		return false
	}
	if s.wildcard {
		return true
	}
	if _, ok := s.repos[repoID]; ok {
		return true
	}
	return false
}

func (s *ScopeSet) Merge(other *ScopeSet) {
	if other.wildcard {
		s.wildcard = true
		return
	}

	for planID := range other.plans {
		s.plans[planID] = struct{}{}
	}
	for repoID := range other.repos {
		s.repos[repoID] = struct{}{}
	}
	for planID := range other.excludedPlans {
		s.excludedPlans[planID] = struct{}{}
	}
	for repoID := range other.excludedRepos {
		s.excludedRepos[repoID] = struct{}{}
	}
}

type PermissionSet struct {
	perms map[v1.Multihost_Permission_Type]ScopeSet
}

func NewPermissionSet(perms []*v1.Multihost_Permission) (*PermissionSet, error) {
	permSet := &PermissionSet{
		perms: make(map[v1.Multihost_Permission_Type]ScopeSet),
	}

	for _, perm := range perms {
		if perm.Scopes == nil {
			continue
		}
		scopeSet, err := NewScopeSet(perm.Scopes)
		if err != nil {
			return nil, err
		}
		permSet.perms[perm.Type] = *scopeSet
	}

	return permSet, nil
}

func (p *PermissionSet) CheckPermissionForPlan(planID string, permType ...v1.Multihost_Permission_Type) bool {
	for _, pt := range permType {
		if scopeSet, ok := p.perms[pt]; ok {
			if scopeSet.ContainsPlan(planID) {
				return true
			}
		}
	}
	return false
}

func (p *PermissionSet) CheckPermissionForRepo(repoID string, permType ...v1.Multihost_Permission_Type) bool {
	for _, pt := range permType {
		if scopeSet, ok := p.perms[pt]; ok {
			if scopeSet.ContainsRepo(repoID) {
				return true
			}
		}
	}
	return false
}
