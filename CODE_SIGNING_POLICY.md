# Code signing policy

Free code signing provided by [SignPath.io](https://about.signpath.io/),
certificate by [SignPath Foundation](https://signpath.org/).

## What is signed

Only `HZD\_EnemyRandomizer\_Map.exe`, built from this repository by the public GitHub Actions
workflow [`.github/workflows/build.yml`](.github/workflows/build.yml). Nothing built on a
personal computer is signed.

## How a release is signed

1. A version tag (for example `v1.1.0`) is pushed to this repository.
2. GitHub Actions builds the exe from that exact commit (Go 1.24.7, `-trimpath`), after
checking that the icon/version resource file is regenerated identically from
`tools/winres`.
3. The unsigned build is submitted to SignPath.
4. An approver manually reviews and approves the signing request.
5. The signed exe is attached to the GitHub release.

## Team roles

* Committers and reviewers: thedotor
* Approvers: [thedotor](https://github.com/YOUR-GITHUB-USERNAME)

All team members use multi-factor authentication for GitHub and SignPath.

## Privacy

See [PRIVACY.md](PRIVACY.md). This program will not transfer any information to other networked
systems unless specifically requested by the user or the person installing or operating it.

