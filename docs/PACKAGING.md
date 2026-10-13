# Packaging

## Tagging a new upstream release

```shell
$ VERSION=X.Y.Z
$ git tag -a -s -m v$VERSION v$VERSION
$ git push origin v$VERSION
```

## Validating downstream packaging

```shell
$ make archive
$ make archive-deps
$ make srpm
$ make rpm [NOCHECK=1]
```
