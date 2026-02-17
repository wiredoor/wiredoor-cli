#!/bin/bash

set -e

if command -v resolvconf > /dev/null; then
  echo "resolvconf command found"
else
  echo "resolvconf not found"
  if [ -f /usr/bin/resolvectl ]; then
    ln -sf /usr/bin/resolvectl /usr/local/bin/resolvconf
  fi
fi