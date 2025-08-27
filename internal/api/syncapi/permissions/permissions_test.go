package permissions

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestPermissionSet_CheckPermissionForPlan(t *testing.T) {
	tests := []struct {
		name        string
		permissions []*v1.Multihost_Permission
		queries     []struct {
			name     string
			permType v1.Multihost_Permission_Type
			planID   string
			expected bool
		}
	}{
		{
			name: "wildcard permissions",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"*"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				planID   string
				expected bool
			}{
				{
					name:     "read plan1 with wildcard",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan1",
					expected: true,
				},
				{
					name:     "read plan2 with wildcard",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan2",
					expected: true,
				},
				{
					name:     "write plan1 without write permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					planID:   "plan1",
					expected: false,
				},
			},
		},
		{
			name: "specific plan permissions",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"plan:plan1", "plan:plan2"},
				},
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					Scopes: []string{"plan:plan1"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				planID   string
				expected bool
			}{
				{
					name:     "read plan1 with specific permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan1",
					expected: true,
				},
				{
					name:     "read plan2 with specific permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan2",
					expected: true,
				},
				{
					name:     "read plan3 without permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan3",
					expected: false,
				},
				{
					name:     "write plan1 with write permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					planID:   "plan1",
					expected: true,
				},
				{
					name:     "write plan2 without write permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					planID:   "plan2",
					expected: false,
				},
			},
		},
		{
			name: "excluded plans with wildcard",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"*", "!plan:secret"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				planID   string
				expected bool
			}{
				{
					name:     "read allowed plan with wildcard and exclusion",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan1",
					expected: true,
				},
				{
					name:     "read excluded plan",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "secret",
					expected: false,
				},
			},
		},
		{
			name: "mixed permissions with exclusions",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"plan:plan1", "plan:plan2", "plan:secret", "!plan:secret"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				planID   string
				expected bool
			}{
				{
					name:     "read plan1 with specific permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan1",
					expected: true,
				},
				{
					name:     "read excluded plan despite explicit permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "secret",
					expected: false,
				},
			},
		},
		{
			name:        "no permissions",
			permissions: []*v1.Multihost_Permission{},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				planID   string
				expected bool
			}{
				{
					name:     "read plan1 with no permissions",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan1",
					expected: false,
				},
			},
		},
		{
			name: "nil scopes permission",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: nil,
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				planID   string
				expected bool
			}{
				{
					name:     "read plan1 with nil scopes",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					planID:   "plan1",
					expected: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permSet, _ := NewPermissionSet(tt.permissions)
			for _, query := range tt.queries {
				t.Run(query.name, func(t *testing.T) {
					result := permSet.CheckPermissionForPlan(query.planID, query.permType)
					if result != query.expected {
						t.Errorf("CheckPermissionForPlan(%v, %q) = %v, want %v", query.permType, query.planID, result, query.expected)
					}
				})
			}
		})
	}
}

func TestPermissionSet_CheckPermissionForRepo(t *testing.T) {
	t.Skip("skipping syncapi tests")
	tests := []struct {
		name        string
		permissions []*v1.Multihost_Permission
		queries     []struct {
			name     string
			permType v1.Multihost_Permission_Type
			repoID   string
			expected bool
		}
	}{
		{
			name: "wildcard permissions",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"*"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				repoID   string
				expected bool
			}{
				{
					name:     "read repo1 with wildcard",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo1",
					expected: true,
				},
				{
					name:     "read repo2 with wildcard",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo2",
					expected: true,
				},
				{
					name:     "write repo1 without write permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					repoID:   "repo1",
					expected: false,
				},
			},
		},
		{
			name: "specific repo permissions",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"repo:repo1", "repo:repo2"},
				},
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					Scopes: []string{"repo:repo1"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				repoID   string
				expected bool
			}{
				{
					name:     "read repo1 with specific permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo1",
					expected: true,
				},
				{
					name:     "read repo2 with specific permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo2",
					expected: true,
				},
				{
					name:     "read repo3 without permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo3",
					expected: false,
				},
				{
					name:     "write repo1 with write permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					repoID:   "repo1",
					expected: true,
				},
				{
					name:     "write repo2 without write permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					repoID:   "repo2",
					expected: false,
				},
			},
		},
		{
			name: "excluded repos with wildcard",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"*", "!repo:sensitive"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				repoID   string
				expected bool
			}{
				{
					name:     "read allowed repo with wildcard and exclusion",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo1",
					expected: true,
				},
				{
					name:     "read excluded repo",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "sensitive",
					expected: false,
				},
			},
		},
		{
			name: "mixed permissions with exclusions",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"repo:repo1", "repo:repo2", "repo:sensitive", "!repo:sensitive"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				repoID   string
				expected bool
			}{
				{
					name:     "read repo1 with specific permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo1",
					expected: true,
				},
				{
					name:     "read excluded repo despite explicit permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "sensitive",
					expected: false,
				},
			},
		},
		{
			name: "mixed repo and plan scopes",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: []string{"repo:repo1", "plan:plan1"},
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				repoID   string
				expected bool
			}{
				{
					name:     "read repo1 with repo permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo1",
					expected: true,
				},
				{
					name:     "read repo2 without repo permission",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo2",
					expected: false,
				},
			},
		},
		{
			name:        "no permissions",
			permissions: []*v1.Multihost_Permission{},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				repoID   string
				expected bool
			}{
				{
					name:     "read repo1 with no permissions",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo1",
					expected: false,
				},
			},
		},
		{
			name: "nil scopes permission",
			permissions: []*v1.Multihost_Permission{
				{
					Type:   v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					Scopes: nil,
				},
			},
			queries: []struct {
				name     string
				permType v1.Multihost_Permission_Type
				repoID   string
				expected bool
			}{
				{
					name:     "read repo1 with nil scopes",
					permType: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
					repoID:   "repo1",
					expected: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permSet, _ := NewPermissionSet(tt.permissions)
			for _, query := range tt.queries {
				t.Run(query.name, func(t *testing.T) {
					result := permSet.CheckPermissionForRepo(query.repoID, query.permType)
					if result != query.expected {
						t.Errorf("CheckPermissionForRepo(%v, %q) = %v, want %v", query.permType, query.repoID, result, query.expected)
					}
				})
			}
		})
	}
}
