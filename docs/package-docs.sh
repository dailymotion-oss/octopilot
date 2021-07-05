#!/bin/bash

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
ROOT_DIR=${DOCS_DIR}/..

mkdir -p ${ROOT_DIR}/dist
mv ${DOCS_DIR}/root/public ${ROOT_DIR}/dist/docs
mv ${DOCS_DIR}/current-version/public ${ROOT_DIR}/dist/docs/${VERSION}
