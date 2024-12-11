package protoutil

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func RepoToRemoteRepo(r *v1.Repo) *v1.RemoteRepo {
	if r == nil {
		return nil
	}
	return &v1.RemoteRepo{
		Id:       r.Id,
		Uri:      r.Uri,
		Password: r.Password,
		Env:      r.Env,
		Flags:    r.Flags,
	}
}

func RemoteRepoToRepo(r *v1.RemoteRepo) *v1.Repo {
	if r == nil {
		return nil
	}
	return &v1.Repo{
		Id:       r.Id,
		Uri:      r.Uri,
		Password: r.Password,
		Env:      r.Env,
		Flags:    r.Flags,
	}
}
