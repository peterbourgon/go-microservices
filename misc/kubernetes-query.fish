#!/usr/bin/env fish

curl -vv -XPOST -d'{"a":"1","b":"2"}' http://localhost:8001/api/v1/proxy/namespaces/default/services/addsvc/concat
