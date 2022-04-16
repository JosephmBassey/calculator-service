#!/bin/bash
NAME=calculator-service
set -o nounset
set -o errexit
set -o errtrace
set -o pipefail
# export GO111MODULE=on
# go get google.golang.org/protobuf/cmd/protoc-gen-go \
#          google.golang.org/grpc/cmd/protoc-gen-go-grpc

source develop.env

# make proto
# We compile the "${NAME}"
GOGC=off go build -v -o "${NAME}" ./cmd/

# declare -a bgpids
function cleanup() {
    set +o errexit
    # comment me if tainted cache is OK and you want to skip the cqlsh port wait time
    # docker container stop --time=0 greypostgres
    rm -f "${NAME}"
}

trap cleanup EXIT

LOGFMT=${LOGFMT:-}
if [ "${LOGFMT}" == "json" ]; then
    ./"${NAME}" | jq -R 'fromjson? | .'
else
    ./"${NAME}"
fi
