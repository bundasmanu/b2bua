#!/bin/bash

set -e

trap "Ok received Exit" HUP INT QUIT TERM

# Extract SIP bind hostname from LOCAL_SIP_ENDPOINT.
# Accepts: host, host:port, udp:host:port, tcp:host:port, ...
dtls_host_from_endpoint() {
    local endpoint="$1"
    local host="$endpoint"

    case "$host" in
        udp:*|tcp:*|tls:*|ws:*|wss:*)
            host="${host#*:}"
            ;;
    esac

    if [[ "$host" == *:* ]]; then
        host="${host%:*}"
    fi

    printf '%s' "$host"
}

# Generate a pod-local DTLS cert when none are mounted (StatefulSet DNS bind).
ensure_dtls_certs() {
    if [[ -n "$DTLS_CERT_FILE" && -n "$DTLS_KEY_FILE" ]]; then
        return 0
    fi

    local dtls_dir="${DTLS_DIR:-/tmp/dtls}"
    local dtls_host

    if [[ -n "$LOCAL_SIP_ENDPOINT" ]]; then
        dtls_host="$(dtls_host_from_endpoint "$LOCAL_SIP_ENDPOINT")"
    else
        dtls_host="127.0.0.1"
    fi

    if [[ -z "$dtls_host" ]]; then
        echo "[ERROR] could not determine DTLS certificate hostname" >&2
        return 1
    fi

    mkdir -p "$dtls_dir"

    echo "[INFO] Generating DTLS certificate for CN/SAN=$dtls_host"
    openssl req -x509 -newkey rsa:2048 -nodes \
        -keyout "$dtls_dir/tls.key" \
        -out "$dtls_dir/tls.crt" \
        -days "${DTLS_CERT_DAYS:-365}" \
        -subj "/CN=$dtls_host" \
        -addext "subjectAltName=DNS:$dtls_host"

    export DTLS_CERT_FILE="$dtls_dir/tls.crt"
    export DTLS_KEY_FILE="$dtls_dir/tls.key"
}

case "$1" in
    shell)
        exec /bin/bash --login
        ;;
    start)
        echo "[INFO] Running...\n"

        ensure_dtls_certs

        ARGS=()
        if [[ -n "$LOCAL_SIP_ENDPOINT" ]]; then
            ARGS+=("--local_addr=$LOCAL_SIP_ENDPOINT")
        fi
        if [[ -n "$OUTBOUND_ADDR" ]]; then
            ARGS+=("--outbound_proxy_addr=$OUTBOUND_ADDR")
        fi
        if [[ -n "$DTLS_CERT_FILE" ]]; then
            ARGS+=("--dtls_cert=$DTLS_CERT_FILE")
        fi
        if [[ -n "$DTLS_KEY_FILE" ]]; then
            ARGS+=("--dtls_key=$DTLS_KEY_FILE")
        fi

        ./b2bua "${ARGS[@]}"
        ;;
    *)
        echo "Executing custom command"
        exec "$@"
        ;;
esac
