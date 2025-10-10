# Release Checklist

Use this checklist before creating a new release to ensure everything is up to date.

## Pre-Release Checklist

### ✅ Code Quality
- [ ] All tests pass: `go test ./...`
- [ ] No race conditions: `go test -race ./...`
- [ ] Linter passes: `golangci-lint run`
- [ ] Go vet passes: `go vet ./...`
- [ ] Build succeeds: `make build`

### ✅ Documentation

#### README.md
- [ ] **Features section** is up to date with new features
- [ ] **Installation instructions** are accurate
- [ ] **Usage examples** reflect current CLI flags
- [ ] **Architecture section** lists all packages correctly
- [ ] **Test coverage table** shows current percentages
- [ ] **Changelog section** includes new version with:
  - [ ] Version number and date
  - [ ] Security improvements (if any)
  - [ ] New features
  - [ ] Performance improvements (if any)
  - [ ] Bug fixes
  - [ ] Breaking changes (if any)
- [ ] **Badges** show correct versions and coverage

#### CLAUDE.md
- [ ] Updated with new patterns or conventions
- [ ] Build commands are current
- [ ] Important notes reflect current state

#### Code Comments
- [ ] All exported functions have godoc comments
- [ ] Complex logic has explanatory comments
- [ ] TODOs are tracked or removed

### ✅ Version Bump

- [ ] Decide version number (following [Semantic Versioning](https://semver.org/)):
  - **MAJOR** (x.0.0) - Breaking changes
  - **MINOR** (1.x.0) - New features, backward compatible
  - **PATCH** (1.2.x) - Bug fixes only
- [ ] Update version references in:
  - [ ] README.md changelog
  - [ ] Any version constants in code
  - [ ] Release notes

### ✅ Testing

- [ ] Test with real nginx logs
- [ ] Test auto-detection feature
- [ ] Test all keyboard controls
- [ ] Test with invalid input (path validation)
- [ ] Test with sample_logs/access.log
- [ ] Verify terminal compatibility

### ✅ Git

- [ ] All changes committed
- [ ] Commit messages follow conventional commits style
- [ ] No uncommitted changes: `git status`
- [ ] On correct branch (typically `main`)
- [ ] Pulled latest changes: `git pull origin main`

## Release Process

1. **Update README.md**
   ```bash
   # Edit README.md changelog section
   # Add new version entry at the top
   git add README.md
   git commit -m "Update README.md for vX.Y.Z release"
   ```

2. **Push changes**
   ```bash
   git push origin main
   ```

3. **Create and push tag**
   ```bash
   # Create annotated tag with release notes
   git tag -a vX.Y.Z -m "Release vX.Y.Z - Brief Description

   ## Features
   - Feature 1
   - Feature 2

   ## Bug Fixes
   - Fix 1
   - Fix 2

   ## Performance
   - Improvement 1
   "

   # Push tag to trigger release workflow
   git push origin vX.Y.Z
   ```

4. **Monitor GitHub Actions**
   - Watch the release workflow build
   - Verify all platforms build successfully
   - Check release artifacts are created

5. **Verify Release**
   - Go to https://github.com/papaganelli/tailnginx/releases
   - Verify release notes are correct
   - Verify binaries are attached
   - Test download and run a binary

## Post-Release

- [ ] Announce release (if applicable)
- [ ] Update any external documentation
- [ ] Close related GitHub issues/milestones
- [ ] Consider creating a blog post or announcement for major versions

## Common Mistakes to Avoid

❌ **Don't:**
- Release without updating README.md first
- Forget to update the changelog
- Skip testing with real data
- Create release tags from unstable branches
- Include debug code or console logs
- Release with failing tests

✅ **Do:**
- Always update README.md before tagging
- Test thoroughly before releasing
- Use semantic versioning correctly
- Write clear, detailed release notes
- Verify GitHub Actions workflow completes

## Emergency Rollback

If you need to rollback a release:

1. Delete the tag locally and remotely:
   ```bash
   git tag -d vX.Y.Z
   git push origin :refs/tags/vX.Y.Z
   ```

2. Delete the GitHub release from the releases page

3. Fix issues, then re-release with the same or higher version number

## Version Number Guide

Use [Semantic Versioning](https://semver.org/):

- **1.0.0** → **2.0.0** - Breaking API changes, major refactors
- **1.0.0** → **1.1.0** - New features, backward compatible
- **1.0.0** → **1.0.1** - Bug fixes only, no new features

### Examples:

- Add new CLI flag: `MINOR` (1.0.0 → 1.1.0)
- Fix crash: `PATCH` (1.0.0 → 1.0.1)
- Remove deprecated feature: `MAJOR` (1.0.0 → 2.0.0)
- Add new dashboard panel: `MINOR` (1.0.0 → 1.1.0)
- Performance improvement: `MINOR` or `PATCH` depending on scope
- Security fix: `PATCH` (but announce it!)
- New package/module: `MINOR` (1.0.0 → 1.1.0)

---

**Last Updated:** 2025-10-10 (for v1.2.0 release)
