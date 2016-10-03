#!/usr/bin/env fish

docker run --name pg -e POSTGRES_USER=tracer -e POSTGRES_PASSWORD=tracer -e POSTGRES_DB=postgres -p 5432:5432 -d postgres
sleep 5
cat $GOPATH/src/github.com/tracer/tracer/storage/postgres/schema.sql | docker run -i --rm --link pg:postgres -e PGPASSWORD=tracer postgres psql -h postgres -U tracer postgres

