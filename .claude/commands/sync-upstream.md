Sync this fork with upstream marcus/sidecar and rebuild.

Steps:
1. Run `git fetch upstream --tags` to get latest upstream changes and version tags
2. Run `git log --oneline HEAD..upstream/main | head -20` to show what's new
3. Run `git merge upstream/main` to merge upstream into the current branch
4. If there are merge conflicts, list them and stop — do NOT auto-resolve
5. Get the latest version tag: `git tag --sort=-v:refname | head -1`
6. Build with version embedded: `go build -ldflags "-X main.Version=<tag>" -o sidecar ./cmd/sidecar`
7. Install: `rm -f ~/.local/bin/sidecar && cp sidecar ~/.local/bin/sidecar`
8. Run `sidecar --version` to confirm the update
9. Show a summary of what changed (new commits pulled, new version)
