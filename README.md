myGoBox
===

A small dropbox like clone written in Go to familiarize my self with the language and using third party libraries


deps 
---

`github.com/aws/aws-sdk-go/aws github.com/radovskyb/watcher`

todo
---
 * add configuration file to unhard code from my env

usage and assumptions 
---

this project assumes you have an aws account and the `aws` cli set up with a `[default]` profile configured

to run:
download deps `go get github.com/aws/aws-sdk-go/aws github.com/radovskyb/watcher` and then build with `go build` in the root of the project, and run the `myGoBox` executable created