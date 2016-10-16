# logspout-oms

An adapter for logspout to write messages to Azure Operations Management Suite.

This folder can be built as a regular docker image with `docker build`. It
uses the Docker file from `github.com/gliderlabs/logspout`. It essentially
copies the included files from this folder on top of the logspout source and
compiles it in a docker container with go installed.

```
import (
	_ "github.com/gliderlabs/logspout/adapters/raw"
	_ "github.com/gliderlabs/logspout/adapters/syslog"
...
	_ "github.com/gliderlabs/logspout/transports/tls"
	_ "github.com/kth/logspout-oms"
)
```

Run it by adding the OMS URL to the command:

```
oms://<workspace-id>.ods.opinsights.azure.com?sharedKey=<urlencoded key>
```

Where workspace-id is the id found under Settings, Connected Sources in
OMS. It's a the alfa-numerical string found there, not the name you gave
the workspace. Key is either of the primary or secondary keys, however
it needs to be manually urlencoded in the URL parameter.

A docker 1.12 swarm mode service to run logspout globally pushing logs
from all containers to OMS can be created like this:

```
docker service create \
  --mode global \
  --restart-condition on-failure \
  --restart-max-attempts 10 \
  --network external_nw \
  --name="logspout" \
  --mount type=bind,src=/var/run/docker.sock,dst=/tmp/docker.sock \
  your-image-tag \
  'oms://your-oms-url-as-specified-above'
```
