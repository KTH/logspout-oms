# logspout-oms

An adapter for logspout to write messages to Azure Operations Management Suite.

## About log formats

This adapter will take lines of output as forwarded by logspout and write them
to OMS according to the following rules:

1. If it is a regular text line, a JSON message will be created in a
   Bunyan-like format (https://github.com/trentm/node-bunyan) and the
   type set to Bunyan. In OMS, such messages will show up as of the
   "Custom Log" type Bunyan_CL. The text will be found in the msg_s
   property in OMS.

   The log level will be set to "ERROR" (level_d = 50) for messages
   printed to stderr from the container, and "INFO" (level_d = 30) for
   messages printed to stdout.

2. If it is a JSON object, the message will be forwarded as is, with
   a "dockerinfo" object with docker meta data added to the structure.
   If the object has a "Type" field, it will be used as the type in the
   OMS request, hence showing up as "MyType_CL" in OMS if set to "MyType".

   If no Type is set, Bunyan is assumed and "Bunyan" will be used
   regardless of the actual JSON object structure for backward
   compatibility.

## Build

This folder can be built as a regular docker image with `docker build`. It
uses the Docker file from `github.com/gliderlabs/logspout`. It essentially
copies the included files from this folder on top of the logspout source and
compiles it in a docker container with go installed.

You can edit modules.go in order to include/exclude modules as you see fit.

## Run

Run it by adding the OMS URL to the command:

```
oms://<workspace-id>.ods.opinsights.azure.com?sharedKey=<urlencoded key>
```

Where workspace-id is the id found under Settings, Connected Sources in
OMS. It's a the alfa-numerical string found there, not the name you gave
the workspace. Key is either of the primary or secondary keys, however
it needs to be manually urlencoded in the URL parameter.

## Swarm global service example

A docker 1.12 swarm mode service to run logspout globally pushing logs
from all containers to OMS can be created like this:

```
docker service create \
  --mode global \
  --restart-condition any \
  --restart-max-attempts 10 \
  --name="logspout" \
  --mount type=bind,src=/var/run/docker.sock,dst=/tmp/docker.sock \
  your-image-tag \
  'oms://your-oms-url-as-specified-above'
```
