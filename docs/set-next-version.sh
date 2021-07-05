#!/bin/bash

echo Next Version: ${VERSION}

# detect this script location (also resolve links since $0 may be a softlink)
PRG="$0"
while [[ -h $PRG ]]; do
  ls=`ls -ld "$PRG"`
  link=`expr "$ls" : '.*-> \(.*\)$'`
  if expr "$link" : '/.*' > /dev/null; then
    PRG="$link"
  else
    PRG=`dirname "$PRG"`/"$link"
  fi
done
DOCS_DIR=$(dirname "$PRG")

# set the next version
yq --inplace eval '.params.octopilot.version = env(VERSION)' $DOCS_DIR/current-version/config.yaml
