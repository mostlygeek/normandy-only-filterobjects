#!/bin/sh

# usage for pretty columns:  ./format-output.sh | column -t

echo "updated created days_ttl revs live? id type slug"
echo "------- ------- -------- ---- ----- -- ---- ----"
awk '{print $6,$5, $7, $8, $4, $1, $2,$3}' $1 | sort -rn
