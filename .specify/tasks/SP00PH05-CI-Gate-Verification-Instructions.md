# SP00PH05 - CI Gate Verification Instructions

**Spec:** SP00 — Environment & Infrastructure Foundation  
**Phase:** 5 - CI Gate Verification  
**Created:** 2026-02-20

---

## Overview

This guide provides explicit instructions for completing the remaining manual tasks in SP00PH05:

1. **SP00PH05T03** - Enable branch protection on GitHub
2. **SP00PH05T04** - Test CI gate with a test PR

---

## SP00PH05T03: Enable Branch Protection

Branch protection ensures that code cannot be merged to the main branch without passing CI checks.

### Option A: Using the PowerShell Script (Recommended)

#### Prerequisites
- GitHub Personal Access Token with `repo` scope
- PowerShell 5.1 or later

#### Steps

1. **Create a GitHub Personal Access Token**
   - Go to: https://github.com/settings/tokens/new
   - Note: Select "Generate new token (classic)"
   - Scopes needed: `repo` (full control of private repositories)
   - Copy the generated token

2. **Run the Branch Protection Script**

   ```powershell
   # Set the token as an environment variable
   $env:GITHUB_TOKEN = "<YOUR_GITHUB_TOKEN>"

   # Run the script
   cd d:\Projects\MUDPuppy
   .\scripts\powershell\setup-branch-protection.ps1 -RepoOwner "amaranth494" -RepoName "MudPuppy"
   ```

3. **Verify Success**
   - The script should output "Branch protection enabled successfully!"
   - Check GitHub: Repo Settings → Branches → Branch protection rules

### Option B: Manual Configuration via GitHub UI

#### Steps

1. **Navigate to Repository Settings**
   - Go to: https://github.com/amaranth494/MudPuppy/settings/branches

2. **Add Branch Protection Rule**
   - Click "Add rule"
   - Branch pattern name: `main`
   - Check "Require status checks to pass before merging"
     - Check "Require branches to be up to date"
     - Select "CI" from the status checks
   - Check "Require pull request reviews before merging"
     - Required approving reviews: 1
   - Check "Include administrators" (optional, leave unchecked for stricter control)
   - Click "Save changes"

3. **Repeat for master branch** (if applicable)
   - Add another rule with branch pattern name: `master`

---

## SP00PH05T04: Test CI Gate

Testing the CI gate ensures that the protection rules work correctly - CI should fail on broken code and pass on valid code.

### Test 1: Verify CI Passes on Valid Code

#### Steps

1. **Create a Test Branch**
   ```bash
   cd d:\Projects\MUDPuppy
   git checkout -b test/ci-gate-test
   ```

2. **Make a Valid Change**
   - Edit any file with a minor change (e.g., add a comment)
   - Commit the change:
     ```bash
     git add .
     git commit -m "test: CI gate test - valid change"
     ```

3. **Push and Create PR**
   ```bash
   git push origin test/ci-gate-test
   ```
   
4. **Create Pull Request**
   - Go to: https://github.com/amaranth494/MudPuppy/compare/main...test/ci-gate-test
   - Click "Create pull request"
   - Title: "Test: CI Gate - Valid Code"
   - Click "Create pull request"

5. **Verify CI Passes**
   - Wait for CI workflow to complete (check the PR page)
   - CI should show a green checkmark
   - If branch protection is enabled, you should see "Merge pull request" button is available

6. **Clean Up**
   - Close the PR (don't merge)
   - Delete the test branch

### Test 2: Verify CI Blocks Merging on Broken Code

#### Steps

1. **Create a Test Branch**
   ```bash
   git checkout -b test/ci-gate-fail
   ```

2. **Make a Breaking Change**
   - Introduce a syntax error in Go code:
   - Edit `cmd/server/main.go` and add invalid syntax, e.g.:
     ```go
     func main() {
         // invalid syntax below
         this is not valid go
     }
     ```
   - Commit the change:
     ```bash
     git add .
     git commit -m "test: CI gate test - breaking change"
     ```

3. **Push and Create PR**
   ```bash
   git push origin test/ci-gate-fail
   ```

4. **Create Pull Request**
   - Go to: https://github.com/amaranth494/MudPuppy/compare/main...test/ci-gate-fail
   - Click "Create pull request"
   - Title: "Test: CI Gate - Broken Code"
   - Click "Create pull request"

5. **Verify CI Fails**
   - Wait for CI workflow to complete
   - CI should show a red X
   - If branch protection is enabled, the "Merge pull request" button should be disabled with message: "Merging is blocked because required status checks haven't completed"

6. **Clean Up**
   - Close the PR (don't merge)
   - Delete the test branch:
     ```bash
     git checkout main
     git push origin --delete test/ci-gate-fail
     ```

---

## Verification Checklist

After completing both tests, verify:

- [ ] CI passes on valid code (green checkmark)
- [ ] CI fails on broken code (red X)
- [ ] Branch protection blocks merging when CI is failing
- [ ] Branch protection allows merging when CI passes

---

## Troubleshooting

### CI Not Running?
- Check: GitHub Actions tab to see workflow runs
- Verify: The workflow file exists at `.github/workflows/ci.yml`

### Branch Protection Not Working?
- Check: Repo Settings → Branches → Branch protection rules
- Verify: "Require status checks to pass before merging" is checked
- Verify: "CI" is selected as a required status check

### Cannot Merge PR?
- This is expected when branch protection is enabled and CI is failing
- Fix the code issues and re-push to update CI status

---

## Next Steps

After completing SP00PH05:

1. Proceed to **SP00PH06** (QA Validation)
2. Continue with **SP00PH07** (Merge & Close)

---

## References

- [GitHub Branch Protection Rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Creating a Personal Access Token](https://github.com/settings/tokens/new)
