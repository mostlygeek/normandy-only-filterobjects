#!/bin/sh

# works on OSX ... should work on linux :)
echo "id || revision count || times filter_expression changed || filter object used || type || last update"
awk '/^[0-9]/ {print $1,$2,$6,$5,$3,$4}' output-1.txt  | column -t  | sort -h | less
