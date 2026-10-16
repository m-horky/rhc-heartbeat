# Packaging

## Tagging a new upstream release

1. Edit the Version value in `rhc-heartbeat.spec`
2. Bump the version in `selinux/rhc_heartbeat.te`
3. Tag the commit and push it

    ```shell
    $ VERSION=X.Y.Z
    $ git add rhc-heartbeat.spec selinux/rhc_heartbeat.te
    $ git commit -s -m "chore: v$VERSION"
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
