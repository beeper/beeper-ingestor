#!/bin/bash
export MAUTRIX_VERSION=$(cat go.mod | grep 'maunium.net/go/mautrix ' | head -n1 | awk '{ print $2 }')
go build -ldflags "-X main.Tag=$(git describe --exact-match --tags 2>/dev/null) -X main.Commit=$(git rev-parse HEAD) -X 'main.BuildTime=`date -Iseconds`' -X 'maunium.net/go/mautrix.GoModVersion=$MAUTRIX_VERSION'" ./cmd/ingestor "$@" || exit 2
