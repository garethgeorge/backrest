package permissions

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

var (
	PermsCanViewResources = []v1.Multihost_Permission_Type{
		v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
		v1.Multihost_Permission_PERMISSION_READ_CONFIG,
		v1.Multihost_Permission_PERMISSION_READ_WRITE_CONFIG,
	}

	PermsCanViewConfiguration = []v1.Multihost_Permission_Type{
		v1.Multihost_Permission_PERMISSION_READ_CONFIG,
		v1.Multihost_Permission_PERMISSION_READ_WRITE_CONFIG,
	}

	PermsCanWriteConfiguration = []v1.Multihost_Permission_Type{
		v1.Multihost_Permission_PERMISSION_READ_WRITE_CONFIG,
	}

	PermsCanViewOperations = []v1.Multihost_Permission_Type{
		v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
	}
)
