# refsafe [![CircleCI](https://circleci.com/gh/Matts966/refsafe.svg?style=svg)](https://circleci.com/gh/Matts966/refsafe) [![codecov](https://codecov.io/gh/Matts966/refsafe/branch/master/graph/badge.svg)](https://codecov.io/gh/Matts966/refsafe)

You can find unsafe use of `reflect` package by using this tool.

```
$ go get -u github.com/Matts966/refsafe/cmd/refsafe
$ refsafe ./path-to-your-project/...
or specify package name like
$ TARGET_PKG=github.com/Matts966/refsafe
$ refsafe $TARGET_PKG/...
```

This tool can be used with `go vet` like
```
$ go vet -vettool=$(which refsafe) 
```

- [x] Ignore functions that do not import `reflect`.
- [x] Check the return value in all the path.
- [ ] Support types that have some variations such as `int32` and `int64`.
