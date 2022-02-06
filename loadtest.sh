#!/bin/bash
url=$1

if [ "$url" = "" ]; then
    echo "Please specify the sitemap url"
    exit
fi

f=$(mktemp)
curl -L "$url" > $f

hurl -conns 100 -discard -loop 100 @"$f"

rm "$f"
