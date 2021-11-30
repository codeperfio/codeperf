package cmd

import "testing"

func Test_fromRemoteToOrgRepo(t *testing.T) {
	type args struct {
		remoteUsed string
	}
	tests := []struct {
		name               string
		args               args
		wantDefaultGitOrg  string
		wantDefaultGitRepo string
	}{
		{"git", args{"git@github.com:codeperfio/pprof-exporter.git"}, "codeperfio", "pprof-exporter"},
		{"git-neg", args{"git@github.com:codeperfio/pprof-exporter"}, "codeperfio", "pprof-exporter"},
		{"git-neg2", args{"github.com:codeperfio/pprof-exporter"}, "codeperfio", "pprof-exporter"},
		{"codeperfio/example-go", args{"git@github.com:codeperfio/example-go.git"}, "codeperfio", "example-go"},
		{"codeperfio/example-go-https", args{"https://github.com/codeperfio/example-go.git"}, "codeperfio", "example-go"},
		{"codeperfio/example-go-https-neg", args{"https://github.com/codeperfio/example-go"}, "codeperfio", "example-go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDefaultGitOrg, gotDefaultGitRepo := fromRemoteToOrgRepo(tt.args.remoteUsed)
			if gotDefaultGitOrg != tt.wantDefaultGitOrg {
				t.Errorf("fromRemoteToOrgRepo() gotDefaultGitOrg = %v, want %v", gotDefaultGitOrg, tt.wantDefaultGitOrg)
			}
			if gotDefaultGitRepo != tt.wantDefaultGitRepo {
				t.Errorf("fromRemoteToOrgRepo() gotDefaultGitRepo = %v, want %v", gotDefaultGitRepo, tt.wantDefaultGitRepo)
			}
		})
	}
}

func Test_getShortHash(t *testing.T) {
	type args struct {
		hash    string
		ndigits int
	}
	tests := []struct {
		name      string
		args      args
		wantShort string
	}{
		{"small", args{"a", 10}, "a"},
		{"7dig", args{"1e872b59013425b7c404a91d16119e8452b983f2", 7}, "1e872b5"},
		{"4dig", args{"1e872b59013425b7c404a91d16119e8452b983f2", 4}, "1e87"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotShort := getShortHash(tt.args.hash, tt.args.ndigits); gotShort != tt.wantShort {
				t.Errorf("getShortHash() = %v, want %v", gotShort, tt.wantShort)
			}
		})
	}
}
