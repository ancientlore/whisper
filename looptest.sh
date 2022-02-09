#!/bin/bash
url=$1

if [ "$url" = "" ]; then
    echo "Please specify the url"
    exit
fi

hurl -addr :9000 -conns 100 -discard -loop 100000 "$url"

