#!/bin/bash

set -e

trap "Ok received Exit" HUP INT QUIT TERM

case "$1" in
    shell)
        exec /bin/bash --login
        ;;
    start)
        echo "[INFO] Running...\n"
        ./b2bua sip:127.0.0.1:5060
        ;;
    *)
        echo "Executing custom command"
        exec "$@"
        ;;
esac
