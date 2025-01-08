package protoutil

import (
	"reflect"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestRepoToRemoteRepo(t *testing.T) {
	tests := []struct {
		name string
		repo *v1.Repo
		want *v1.RemoteRepo
	}{
		{
			name: "basic conversion",
			repo: &v1.Repo{
				Id:       "1",
				Uri:      "http://example.com",
				Password: "password",
				Env:      []string{"FOO=BAR"},
				Flags:    []string{"flag1", "flag2"},
			},
			want: &v1.RemoteRepo{
				Id:       "1",
				Uri:      "http://example.com",
				Password: "password",
				Env:      []string{"FOO=BAR"},
				Flags:    []string{"flag1", "flag2"},
			},
		},
		{
			name: "empty repo",
			repo: &v1.Repo{},
			want: &v1.RemoteRepo{},
		},
		{
			name: "nil repo",
			repo: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RepoToRemoteRepo(tt.repo); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RepoToRemoteRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoteRepoToRepo(t *testing.T) {
	tests := []struct {
		name       string
		remoteRepo *v1.RemoteRepo
		want       *v1.Repo
	}{
		{
			name: "basic conversion",
			remoteRepo: &v1.RemoteRepo{
				Id:       "1",
				Uri:      "http://example.com",
				Password: "password",
				Env:      []string{"FOO=BAR"},
				Flags:    []string{"flag1", "flag2"},
			},
			want: &v1.Repo{
				Id:       "1",
				Uri:      "http://example.com",
				Password: "password",
				Env:      []string{"FOO=BAR"},
				Flags:    []string{"flag1", "flag2"},
			},
		},
		{
			name:       "empty remote repo",
			remoteRepo: &v1.RemoteRepo{},
			want:       &v1.Repo{},
		},
		{
			name:       "nil remote repo",
			remoteRepo: nil,
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoteRepoToRepo(tt.remoteRepo); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoteRepoToRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}
