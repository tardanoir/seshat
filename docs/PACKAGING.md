# Packaging & Releases

Releases are fully automated. Pushing a tag `vX.Y.Z` triggers
`.github/workflows/release.yml`, which runs GoReleaser to build binaries,
create the GitHub release, and publish to every install channel below.

```sh
git tag v1.0.1
git push origin v1.0.1
```

## Install channels

| Channel | User command | Status |
|---|---|---|
| Install script | `curl -fsSL https://raw.githubusercontent.com/tardanoir/seshat/main/install.sh \| sh` | ready |
| `go install` | `go install github.com/tardanoir/seshat@latest` | ready |
| Homebrew | `brew install tardanoir/tap/seshat` | needs tap repo |
| AUR (Arch) | `yay -S seshat-bin` | needs one-time bootstrap |
| Scoop (Windows) | `scoop bucket add tardanoir https://github.com/tardanoir/scoop-bucket && scoop install seshat` | needs bucket repo |
| apt / dnf | see Cloudsmith below | needs Cloudsmith account |
| Manual `.deb`/`.rpm` | `sudo dpkg -i seshat_*.deb` / `sudo rpm -i seshat_*.rpm` | ready (release assets) |

## One-time setup

### Required GitHub secrets

| Secret | Used for | How to get it |
|---|---|---|
| `TAP_GITHUB_TOKEN` | Pushing to `homebrew-tap` and `scoop-bucket` | A classic PAT with `repo` scope (the default `GITHUB_TOKEN` can't push to other repos). |
| `AUR_KEY` | Publishing `seshat-bin` to the AUR | SSH private key whose public half is on your AUR account. |
| `CLOUDSMITH_API_KEY` | apt/dnf repo | Cloudsmith account API key. Optional — the step is skipped if unset. |

### Homebrew & Scoop buckets

Create two empty public repos under your account; GoReleaser commits the
manifests on each release:

- `tardanoir/homebrew-tap`
- `tardanoir/scoop-bucket`

### AUR bootstrap (one time)

The AUR package repo must exist before the first automated release:

1. Create an AUR account and add an SSH **public** key to your profile.
2. Put the matching **private** key in the `AUR_KEY` GitHub secret.
3. Initialize the repo once:

   ```sh
   git clone ssh://aur@aur.archlinux.org/seshat-bin.git
   cd seshat-bin
   # add a minimal PKGBUILD + .SRCINFO, then:
   git add PKGBUILD .SRCINFO
   git commit -m "Initial import"
   git push
   ```

   After this, GoReleaser keeps it updated on every release.

### Cloudsmith (apt / dnf repo)

1. Create a free open-source repo at cloudsmith.io named `seshat`.
2. Add the API key as the `CLOUDSMITH_API_KEY` secret.
3. In `.github/workflows/release.yml`, replace `any-distro/any-version`
   with the distros you support (e.g. `debian/bookworm`, `ubuntu/jammy`,
   `el/9`). Push one entry per distro you want available.
4. Users then run the setup script Cloudsmith generates, e.g.:

   ```sh
   curl -1sLf 'https://dl.cloudsmith.io/public/tardanoir/seshat/setup.deb.sh' | sudo bash
   sudo apt install seshat
   ```

## Local dry run

```sh
goreleaser release --snapshot --clean   # builds into dist/ without publishing
goreleaser check                        # validate .goreleaser.yaml
```
