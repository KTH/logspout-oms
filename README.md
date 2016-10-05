# logspout-oms
An adapter for logspout to write messages to Azure Operations Management Suite

Use by importing `github.com/kth/logspout-oms` in `modules.go` of logspout.

```
import (
	_ "github.com/gliderlabs/logspout/adapters/raw"
	_ "github.com/gliderlabs/logspout/adapters/syslog"
...
	_ "github.com/gliderlabs/logspout/transports/tls"
	_ "github.com/kth/logspout-oms"
)
```

Then build your image.

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
  --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
  your-image-tag \
  'oms://your-oms-url-as-specified-above'
```
