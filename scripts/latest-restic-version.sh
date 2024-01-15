#!/bin/bash

curl -s https://api.github.com/repos/restic/restic/releases/latest \
| grep "https://api.github.com/repos/restic/restic/tarball/" \
| sed -E 's/.*v([0-9]+\.[0-9]+\.[0-9]+).*/\1/

