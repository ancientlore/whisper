#!/bin/bash
url=$1

if [ "$url" = "" ]; then
    echo "Please specify the sitemap url"
    exit
fi

f=$(mktemp)
curl -L "$url" > $f

hurl -addr :9000 -conns 10 -discard -loop 100000 @"$f"

rm "$f"
