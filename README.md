# Iron-App

`iron-app` extends [envconsul][1] with the
following features, described details below.

* Service discovery, following [Twelve-Factors app][3]
* App logs hooks
* App performances storage
* App metadata storage in consul

## Download & Usage

```
go get github.com/hivetech/iron-app
```

Run `iron-app` to see the usage help:

```
$ iron-app
Usage: iron-app [options] prefix child...

  Exoskeleton for applications.

  Load env from Consul's K/V store, discover provided services, route
  application output and store performances and application metadata.

Options:

  -addr="127.0.0.1:8500": consul HTTP API address with port
  -dc="": consul datacenter, uses local if blank
  -discover="": Comma separated <service:tag> on the network to discover
  -errexit=false: exit if there is an error watching config keys
  -loghook="": An app where to send logs [pushbullet]
  -reload=false: if set, restarts the process when config changes
  -sanitize=true: turn invalid characters in the key into underscores
  -upcase=true: make all environmental variable keys uppercase
  -verbose=false: Extend log output to debug level
```

## Process env injection

`iron-app` replicates the full possibilities of [envconsul][1].

## Service discovery

You can ask iron-app to automatically inject into process environment how to
access a remote service, known by consul.

Considering an application registered in the console network under `redis`
service with a tag `cache` :

```
$ ./iron-app --discover redis:cache app/env env
...
INFO[0000] env REDIS_HOST=172.17.0.4
INFO[0000] env REDIS_PORT=80
INFO[0000] Done
```

## Performances monitoring

TODO

## Log hooks

`iron-app` outputs logs on `stdout` and `stderr` but it also comes with
built-in routines that ship them elsewhere :

* File - `--loghook anything.log`
* [Hipchat](http://hipchat.com/) - Export `HIPCHAT_API_KEY` and `HIPCHAT_ROOM` and use `--loghook hipchat`
* [Pushbullet](http://pushbullet.com/) - Export `PUSHBULLET_API_KEY` and `PUSHBULLET_DEVICE` and use `--loghook pushbullet`

Currently, hipchat and pushbullet catch only `panic`, `fatal` and `error`
levels as configured in `log/hipchat.go` and `log/pushbullet.go`.


[1]: https://github.com/hashicorp/envconsul
[3]: http://12factor.net/
