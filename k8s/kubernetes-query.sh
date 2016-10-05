#!/usr/bin/env sh

curl -vv -XPOST -d'{"a":"Foo","b":"bar"}' http://localhost:8001/api/v1/proxy/namespaces/default/services/addsvc/concat
