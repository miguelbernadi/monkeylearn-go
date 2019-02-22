# Contributing

## Issue guidelines

## Commit requirements

Commits should have a summary and a body. The summary should be
descriptive of the commit's intent, use active voice and start with a
verb. It's length should not exceed 50 characters and must not exceed
72 characters.

The body can be as long as needed and explained the rationale behind
the change, not describe the changes inside the commit themselves. It
should provide all information needed to understand the commit in
isolation.

Commits should include a `Signed-off` entry to conform with the
Developer Certificate of Origin, which effectively states the
contribution is done in good faith under the current license of the
project. The license terms can be reviewed in the [LICENSE](LICENSE)
file in the repo, and the requirements of the DCO are in the
[DCO](DCO) file in the repo.

```
Add an example commit

This commit is done here to show what a typical message looks
like. Here I describe the reasoning behind the change, which is to show
what a commit should look like.

Signed-off-by: Miguel Bernabeu <miguel.bernabeu@lobber.eu>
```

This line will be added by `git` automatically if you add the `-s` flag when creating a commit:
```
git commit -s
```

## Pull Request requirements
