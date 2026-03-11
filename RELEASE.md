# Release workflow

- Bump version in `Makefile`.
- Update `CHANGELOG.md` with `make update-changelog`.
- Merge PR.
- Tag version in main branch: `make tag`

## Upgrade dependencies

```bash
go get -u
```
