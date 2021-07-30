#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

root=$(dirname "${BASH_SOURCE[0]}")/..
kustomize="kustomize"
flavors_dir="${root}/templates/flavors/"
overlays_dir="${root}/overlays/clusters/"
mkdir -p "${flavors_dir}"
find "${overlays_dir}"* -maxdepth 0 -type d -print0 | \
  xargs -0 -I {} basename {} | \
  grep -v patches | \
  xargs -I {} sh -c "${kustomize} build --load-restrictor LoadRestrictionsNone --reorder none ${overlays_dir}{} > ${flavors_dir}cluster-template-{}.yaml"