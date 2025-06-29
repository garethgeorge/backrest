package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/api/syncapi/permissions"
	"github.com/garethgeorge/backrest/internal/config/validationutil"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func ValidateConfig(c *v1.Config) error {
	var err error

	if e := validationutil.ValidateID(c.Instance, validationutil.IDMaxLen); e != nil {
		if errors.Is(e, validationutil.ErrEmpty) {
			zap.L().Warn("ACTION REQUIRED: instance ID is empty, will be required in a future update. Please open the backrest UI to set a unique instance ID. Until fixed this warning (and related errors) will print periodically.")
		} else {
			err = multierror.Append(err, fmt.Errorf("instance ID %q invalid: %w", c.Instance, e))
		}
	}

	if e := validateAuth(c.Auth); e != nil {
		err = multierror.Append(err, fmt.Errorf("auth: %w", e))
	}

	repos := make(map[string]*v1.Repo)
	if c.Repos != nil {
		for _, repo := range c.Repos {
			if e := validateRepo(repo); e != nil {
				err = multierror.Append(err, fmt.Errorf("repo %s: %w", repo.GetId(), e))
			}
			if _, ok := repos[repo.Id]; ok {
				err = multierror.Append(err, fmt.Errorf("repo %s: duplicate id", repo.GetId()))
			}
			repos[repo.Id] = repo
		}
		slices.SortFunc(c.Repos, func(a, b *v1.Repo) int {
			if a.Id < b.Id {
				return -1
			}
			return 1
		})
	}

	if c.Plans != nil {
		plans := make(map[string]*v1.Plan)
		for _, plan := range c.Plans {
			if _, ok := plans[plan.Id]; ok {
				err = multierror.Append(err, fmt.Errorf("plan %s: duplicate id", plan.GetId()))
			}
			plans[plan.Id] = plan
			if e := validatePlan(plan, repos); e != nil {
				err = multierror.Append(err, fmt.Errorf("plan %s: %w", plan.GetId(), e))
			}
		}
		slices.SortFunc(c.Plans, func(a, b *v1.Plan) int {
			if a.Id < b.Id {
				return -1
			}
			return 1
		})
	}

	if e := validateMultihost(c); e != nil {
		err = multierror.Append(err, fmt.Errorf("multihost: %w", e))
	}

	return err
}

func validateRepo(repo *v1.Repo) error {
	var err error

	if e := validationutil.ValidateID(repo.Id, 0); e != nil {
		err = multierror.Append(err, fmt.Errorf("id %q invalid: %w", repo.Id, e))
	}

	// GUID or AutoInitialize must be set, but not both.
	if repo.Guid != "" {
		if repo.AutoInitialize {
			err = multierror.Append(err, fmt.Errorf("auto_initialize set with guid but guid implies that repo is already initialized"))
		}
		if len(repo.Guid) != 64 { // 64 bits in hex
			err = multierror.Append(err, fmt.Errorf("guid %q invalid: must be 64 characters", repo.Guid))
		}
	} else if !repo.AutoInitialize {
		err = multierror.Append(err, fmt.Errorf("guid is required unless using auto_initialize to implicitly initialize repos"))
	}

	if repo.Uri == "" {
		err = multierror.Append(err, errors.New("uri is required"))
	}

	if repo.PrunePolicy.GetSchedule() != nil {
		if e := protoutil.ValidateSchedule(repo.PrunePolicy.GetSchedule()); e != nil {
			err = multierror.Append(err, fmt.Errorf("prune policy schedule: %w", e))
		}
	}

	if repo.CheckPolicy.GetSchedule() != nil {
		if e := protoutil.ValidateSchedule(repo.CheckPolicy.GetSchedule()); e != nil {
			err = multierror.Append(err, fmt.Errorf("check policy schedule: %w", e))
		}
	}

	for _, env := range repo.Env {
		if !strings.Contains(env, "=") {
			err = multierror.Append(err, fmt.Errorf("invalid env var %s, must take format KEY=VALUE", env))
		}
	}

	slices.Sort(repo.Env)

	return err
}

func validatePlan(plan *v1.Plan, repos map[string]*v1.Repo) error {
	var err error
	if e := validationutil.ValidateID(plan.Id, 0); e != nil {
		err = multierror.Append(err, fmt.Errorf("id %q invalid: %w", plan.Id, e))
	}

	if plan.Schedule != nil {
		if e := protoutil.ValidateSchedule(plan.Schedule); e != nil {
			err = multierror.Append(err, fmt.Errorf("backup schedule: %w", e))
		}
	}

	for idx, p := range plan.Paths {
		if p == "" {
			err = multierror.Append(err, fmt.Errorf("path[%d] cannot be empty", idx))
		}
	}

	if plan.Repo == "" {
		err = multierror.Append(err, fmt.Errorf("repo is required"))
	}

	if _, ok := repos[plan.Repo]; !ok {
		err = multierror.Append(err, fmt.Errorf("repo %q not found", plan.Repo))
	}

	if plan.Retention != nil && plan.Retention.Policy == nil {
		err = multierror.Append(err, errors.New("retention policy must be nil or must specify a policy"))
	} else if policyTimeBucketed, ok := plan.Retention.GetPolicy().(*v1.RetentionPolicy_PolicyTimeBucketed); ok {
		if proto.Equal(policyTimeBucketed.PolicyTimeBucketed, &v1.RetentionPolicy_TimeBucketedCounts{}) {
			err = multierror.Append(err, errors.New("time bucketed policy must specify a non-empty bucket"))
		}
	}

	slices.Sort(plan.Paths)

	return err
}

func validateAuth(auth *v1.Auth) error {
	if auth == nil || auth.Disabled {
		return nil
	}

	if len(auth.Users) == 0 {
		return errors.New("auth enabled but no users")
	}

	for _, user := range auth.Users {
		if e := validationutil.ValidateID(user.Name, 0); e != nil {
			return fmt.Errorf("user %q: %w", user.Name, e)
		}
		if user.GetPasswordBcrypt() == "" {
			return fmt.Errorf("user %q: password is required", user.Name)
		}
	}

	return nil
}

func validateMultihost(config *v1.Config) (err error) {
	multihost := config.GetMultihost()
	if multihost == nil {
		return
	}

	if multihost.GetIdentity().GetKeyid() == "" {
		return errors.New("identity keyid is required")
	}

	if _, err := cryptoutil.NewPrivateKey(multihost.GetIdentity()); err != nil {
		return fmt.Errorf("verify private key: %w", err)
	}

	seenInstanceIDs := make(map[string]struct{})
	seenInstanceIDs[config.Instance] = struct{}{}
	assertIDNew := func(id string) error {
		if _, ok := seenInstanceIDs[id]; ok {
			return fmt.Errorf("instance ID %q is already used by another peer, an instance ID can only appear once as either a known host OR authorized client of the instance", id)
		}
		seenInstanceIDs[id] = struct{}{}
		return nil
	}

	for _, peer := range multihost.GetAuthorizedClients() {
		if e := validatePeer(peer, false); e != nil {
			err = multierror.Append(err, fmt.Errorf("authorized client %q: %w", peer.GetInstanceId(), e))
		}
		if e := assertIDNew(peer.GetInstanceId()); e != nil {
			err = multierror.Append(err, fmt.Errorf("authorized client %q: %w", peer.GetInstanceId(), e))
		}
	}

	for _, peer := range multihost.GetKnownHosts() {
		if e := validatePeer(peer, true); e != nil {
			err = multierror.Append(err, fmt.Errorf("known host %q: %w", peer.GetInstanceId(), e))
		}
		if e := assertIDNew(peer.GetInstanceId()); e != nil {
			err = multierror.Append(err, fmt.Errorf("known host %q: %w", peer.GetInstanceId(), e))
		}
	}

	return
}

func validatePeer(peer *v1.Multihost_Peer, isKnownHost bool) error {
	if e := validationutil.ValidateID(peer.InstanceId, validationutil.IDMaxLen); e != nil {
		return fmt.Errorf("id %q invalid: %w", peer.InstanceId, e)
	}

	if peer.GetKeyid() == "" {
		return errors.New("keyid must be specified")
	}

	if isKnownHost {
		if peer.InstanceUrl == "" {
			return errors.New("instance URL is required for known hosts")
		}
	}

	if peer.KeyidVerified && peer.GetKeyid() == "" {
		return errors.New("public key cannot be marked as verified if it is unset, the keyid must be specified at a minimum")
	}

	_, err := permissions.NewPermissionSet(peer.GetPermissions())
	if err != nil {
		return fmt.Errorf("peer permissions: %w", err)
	}

	return nil
}
