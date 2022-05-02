#!/usr/bin/env bash
# shell wrapper runs a shell script inside of the devbase
# repository, which contains shared shell scripts.
set -e

echo "Updating plugin.yaml version to $1"

currentVersion="$(yq -r '.version' ./plugin.yaml)"

sed s/"$currentVersion"/"$1"/ plugin.yaml >plugin.yaml.tmp

mv plugin.yaml.tmp plugin.yaml

#
