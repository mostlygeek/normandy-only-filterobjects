#!/bin/sh

# usage for pretty columns:  ./format-output.sh | column -t

echo "end <- start days_ttl revs live? id type slug"
awk '{print $6,"<-",$5, $7, $8, $4, $1, $2,$3}' $1 | sort -rn
