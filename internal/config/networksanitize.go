package config

import (
	"slices"
	"sort"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/proto"
)

func SanitizeForNetwork(config *v1.Config) *v1.Config {
	clone := proto.Clone(config).(*v1.Config)

	// Sanitize the Multihost.Identity field to only include the Keyid
	if clone.Multihost != nil && clone.Multihost.Identity != nil {
		clone.Multihost.Identity = sanitizePrivateKey(clone.Multihost.Identity)
	}

	// Sanitize the users password hashes
	for _, user := range clone.GetAuth().GetUsers() {
		if user.GetPassword() != nil {
			user.Password = &v1.User_PasswordBcrypt{
				PasswordBcrypt: "********",
			}
		}
	}

	return clone
}

func RehydrateNetworkSanitizedConfig(sanitized *v1.Config, full *v1.Config) *v1.Config {
	clone := proto.Clone(sanitized).(*v1.Config)

	// Rehydrate the Multihost.Identity field with the full private key
	if clone.Multihost != nil && clone.Multihost.Identity != nil {
		clone.Multihost.Identity = proto.Clone(full.Multihost.Identity).(*v1.PrivateKey)
		proto.Merge(clone.Multihost.Identity, sanitized.Multihost.Identity)
	}

	// Loop over the full config users and rehydrate the clone users
	fullUsers := slices.Clone(full.GetAuth().GetUsers())
	sanitizedUsers := clone.GetAuth().GetUsers()
	slices.SortFunc(fullUsers, func(a, b *v1.User) int {
		return strings.Compare(a.GetName(), b.GetName())
	})
	for i, user := range sanitizedUsers {
		if user.GetPasswordBcrypt() == "********" {
			index := sort.Search(len(fullUsers), func(j int) bool {
				return fullUsers[j].GetName() >= user.GetName()
			})
			if index < len(fullUsers) && fullUsers[index].GetName() == user.GetName() {
				sanitizedUsers[i] = proto.Clone(fullUsers[index]).(*v1.User)
			}
		}
	}

	return clone
}

func sanitizePrivateKey(proto *v1.PrivateKey) *v1.PrivateKey {
	return &v1.PrivateKey{
		Keyid: proto.Keyid,
	}
}
