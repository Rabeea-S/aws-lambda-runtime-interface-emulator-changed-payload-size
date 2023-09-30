#!/bin/sh
# Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.

if [ $# -ne 1 ]; then
  echo "entrypoint requires the handler name to be the first argument" 1>&2
  exit 142
fi
export _HANDLER="$1"

useradd -M user
su -c "chmod -R 777 /home" user
su -c "chmod -R 777 /tmp" user

RUNTIME_ENTRYPOINT=/var/runtime/bootstrap
if [ -z "${AWS_LAMBDA_RUNTIME_API}" ]; then
  exec sudo -u user /usr/local/bin/aws-lambda-rie $RUNTIME_ENTRYPOINT
else
  exec sudo -u user $RUNTIME_ENTRYPOINT
fi
