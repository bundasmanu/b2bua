#!/bin/bash

set -e

trap "Ok received Exit" HUP INT QUIT TERM

case "$1" in
    shell)
        exec /bin/bash --login
        ;;
    start)
        echo "[INFO] Running...\n"

        ARGS=()
        if [[ -n "$LOCAL_SIP_ENDPOINT" ]]; then
            ARGS+=("--local_addr=$LOCAL_SIP_ENDPOINT")
        fi
        if [[ -n "$OUTBOUND_ADDR" ]]; then
            ARGS+=("--outbound_proxy_addr=$OUTBOUND_ADDR")
        fi

        ./b2bua "${ARGS[@]}"
        ;;
    *)
        echo "Executing custom command"
        exec "$@"
        ;;
esac
