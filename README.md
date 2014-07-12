# envconsul

envconsul sets environmental variables for processes by reading them
from [Consul's K/V store](http://www.consul.io).

envconsul allows applications to be configured with environmental variables,
without having to be knowledgable about the existence of Consul. This makes
it especially easy to configure applications throughout all your
environments: development, testing, production, etc.

envconsul is inspired by [envdir](http://cr.yp.to/daemontools/envdir.html)
in its simplicity, name, and function.

## Download & Usage

Download a release from the
[releases page](#).
Run `envconsul` to see the usage help:

```

$ envconsul
Usage: envconsul [options] prefix child...

  Sets environmental variables for the child process by reading
  K/V from Consul's K/V store with the given prefix.

Options:

  -addr="127.0.0.1:8500": consul HTTP API address with port
  -dc="": consul datacenter, uses local if blank
  -errexit=false: exit if there is an error watching config keys
  -logfile="": If provided, redirect logs to this file
  -reload=false: if set, restarts the process when config changes
  -sanitize=false: turn invalid characters in the key into underscores
  -upcase=false: make all environmental variable keys uppercase
  -verbose=false: Extend log output to debug level
  -web="": Comma separated web services on the network to discover
```

## Example

We run the example below against our
[NYC demo server](http://nyc1.demo.consul.io). This lets you set
keys/values in a public place to just quickly test envconsul. Note
that the demo server will clear the k/v store every 30 minutes.

After setting the `prefix/FOO` key to "bar" on the demo server,
we can see it work:

```
$ envconsul -addr="nyc1.demo.consul.io:80" prefix env
INFO[0000] env FOO=bar
```

We can also ask envconsul to watch for any configuration changes
and restart our process:

```
$ envconsul -addr="nyc1.demo.consul.io:80" -reload \
  prefix /bin/sh -c "env; echo "-----"; sleep 1000"
INFO[0000] env FOO=bar
INFO[0000] env -----
INFO[0000] env FOO=baz
INFO[0000] env -----
INFO[0000] env FOO=baz
INFO[0000] env BAR=FOO
INFO[0000] env -----
```

The above output happened by setting keys and values within
the online demo UI while envconsul was running.

## Service discovery

You can ask envconsul to automatically inject into process environment how to
access a remote service, known by consul.

Considering an application registered in the console network under `redis`
service with a tag `cache` :

```
$ ./envconsul --discover redis:cache app/env env
...
INFO[0000] env REDIS_HOST=172.17.0.4
INFO[0000] env REDIS_PORT=80
INFO[0000] Done
```

## Log hooks

*envconsul* outputs logs on `stdout` and `stderr` but it also comes with
built-in routines that ship them elsewhere :

* File - `--loghook anything.log`
* [Hipchat](http://hipchat.com/) - Use `--loghook hipchat` and export `HIPCHAT_API_KEY` and `HIPCHAT_ROOM`
* [Pushbullet](http://pushbullet.com/) - Use `--loghook pushbullet` and export `PUSHBULLET_API_KEY` and `PUSHBULLET_DEVICE`

Currently, hipchat and pushbullet catch only `panic`, `fatal` and `error`
levels as configured in `log/hipchat.go` and `log/pushbullet.go`.
