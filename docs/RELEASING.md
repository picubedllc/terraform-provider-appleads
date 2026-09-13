# Releasing

GitHub Releases for this provider are built and GPG-signed by GoReleaser when a
semantic version tag is pushed. The Terraform Registry requires that checksums
file to be signed with the public key registered for `picubedllc/appleads`.

Do **not** cut `v0.1.0` until [PI-29](https://linear.app/picubed/issue/PI-29/publish-terraform-registry-provider)
is unblocked. Registering the provider and signing key with HashiCorp is fine;
publishing a version is not.

## Versioning

Tags must be Terraform Registry semver with a `v` prefix:

```
vMAJOR.MINOR.PATCH
vMAJOR.MINOR.PATCH-prerelease
```

Examples: `v0.1.0`, `v0.1.1`, `v1.0.0`, `v0.2.0-alpha.1`.

Prereleases are installable only when a user pins them. `terraform init` will
not pick them as "latest".

There must not be a branch with the same name as the tag. Never retag or
replace assets on an already-published version; ship a new patch instead.

## Cut a release

1. Merge the work into `main`. The tag must point at a commit that is on `origin/main`.
2. From a clean checkout of that commit:

   ```bash
   git checkout main
   git pull --ff-only
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. GitHub Actions runs the `Release` workflow against the `release` environment.
   Approve the environment deployment if asked.
4. Confirm the GitHub Release has zips, `*_SHA256SUMS`, `*_SHA256SUMS.sig`, and
   `*_manifest.json`.
5. After the provider is published on the Registry, a GitHub `release` webhook
   notifies HashiCorp. Future tags ingress automatically.

## Where the signing key lives

| Item | Location |
| --- | --- |
| Public key | Terraform Registry signing keys for the `picubedllc` namespace |
| Private key | GitHub Environment **`release`**, secret `GPG_PRIVATE_KEY` (ASCII-armored, including BEGIN/END lines) |
| Passphrase | Same environment, secret `PASSPHRASE` |

Do **not** put these in repository secrets, in `appleads-integration`, in the
repo, or in docs. Environment secrets are only available to jobs that declare
`environment: release`. That environment is limited to tags matching `v*` and
requires a reviewer.

The HashiCorp scaffolding names (`GPG_PRIVATE_KEY`, `PASSPHRASE`) are required
by `.github/workflows/release.yml`.

The Registry accepts RSA or DSA keys, not the default ECC type.

## Security notes

GitHub Environment secrets are encrypted at rest and are the right place for
this key — better than repository secrets, because fork PRs and unrelated
workflows cannot read them.

They are still CI credentials. Anyone who can change workflows on `main` and
get a `release` job to run can use the key to sign. Protection on that
environment (reviewers + tag-only deployments) is what keeps that from being
an unattended leak.

Do not mix the signing key with Apple Ads API credentials. Live tests use
`appleads-integration`; signing uses `release`.

## Rotation and escrow

If the key is compromised or needs rotation:

1. Generate a new RSA GPG key (not ECC).
2. Add the new public key in Terraform Registry → Signing Keys. Keep the old
   public key listed so historical versions still verify.
3. Replace `GPG_PRIVATE_KEY` and `PASSPHRASE` on the `release` environment.
4. Cut a **new** provider version. Do not rewrite an old GitHub Release.

Keep an escrow copy of the private key and passphrase in the Pi Cubed password
manager, available to at least one other maintainer besides whoever generated
it. Losing the only copy means old versions stay verifiable (public key is on
the Registry) but you cannot sign new ones until you rotate.

## Local dry-run (unsigned)

```bash
make release-snapshot
```

This does not sign or publish. Signed releases only happen from the `Release`
workflow.
