#!/bin/bash

set -e

echo "executing postinstall"

if command -v resolvconf > /dev/null; then
  echo "resolvconf command found"
else
  echo "resolvconf not found"
  ln -s /usr/bin/resolvectl /usr/local/bin/resolvconf
fi