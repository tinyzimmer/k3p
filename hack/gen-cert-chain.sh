#!/bin/bash

set -e
set -o pipefail

usage () {
    echo "USAGE: ${0} [-d outdir] [-l chain_length]"
    exit 1
}

main () {
    while getopts ":d:l:h" opt ; do
        case ${opt} in
            d )
                destination="${OPTARG}"
                ;;
            l )
                chain_length="${OPTARG}"
                ;;
            h )
                usage
                ;;
            \? )
                echo "Invalid option: ${OPTARG}" 1>&2
                usage
                ;;
            : )
                echo "Invalid option: ${OPTARG} requires an argument" 1>&2
                usage
                ;;
        esac
    done
    if [[ -z "${destination}" ]] ; then
        destination="$(pwd)/tls"
    fi
    if [[ -z "${chain_length}" ]] ; then
        chain_length=3
    fi

    re='^[0-9]+$'
    if ! [[ ${chain_length} =~ $re ]] ; then
        echo "error: ${chain_length} is not a number" 1>&2
        exit 2
    fi

    echo "Writing chain to ${destination} with a depth of ${chain_length}"
    mkdir -p "${destination}"

    set -x

    trap "rm -f ${destination}/*.srl ; rm -f ${destination}/*.csr" EXIT

    gen-ca "${destination}"
    fill-cert-chain "${destination}" "${chain_length}"
}

gen-ca () {
    local outdir="${1}"
    local keypath="${outdir}/ca.key"
    local csrpath="${outdir}/ca.csr"
    local certpath="${outdir}/ca.crt"

    echo "Generating CA private key"
    openssl genrsa -out "${keypath}" 4096
    echo "Generating CA CSR"
    openssl req -new -addext basicConstraints=CA:TRUE -sha256 -key "${keypath}" -out "${csrpath}" -subj "/CN=TEST-CA"
    echo "Self signing CA certificate for 10 years"
    openssl x509 -req -extfile <(printf "keyUsage=critical,cRLSign,digitalSignature,keyCertSign\nbasicConstraints=critical,CA:TRUE\nextendedKeyUsage=serverAuth\nsubjectAltName=DNS:kubenab.kube-system.svc,DNS:localhost") -signkey "${keypath}" -in "${csrpath}" -req -days 3650 -out "${certpath}"
}

fill-cert-chain() {
    local outdir="${1}"
    local chain_length="${2}"


    let "num_certs = ${chain_length} - 1"

    local signercert="${outdir}/ca.crt"
    local signerkey="${outdir}/ca.key"

    for cert_num in $(seq 1 ${num_certs}) ; do
        if [[ ${cert_num} == ${num_certs} ]] ; then 
            name="leaf"
        else
            name="intermediate-${cert_num}"
        fi

        local keypath="${outdir}/${name}.key"
        local csrpath="${outdir}/${name}.csr"
        local certpath="${outdir}/${name}.crt"

        echo "Generating private key for ${name}"
        openssl genrsa -out "${keypath}" 4096
        echo "Generate CSR for ${name}"
        openssl req -new -sha256 -key "${keypath}" -out "${csrpath}" -subj "/CN=TEST-${name^^}" -addext "subjectAltName=DNS:kubenab.kube-system.svc,DNS:localhost"
        echo "Signing certificate for ${name}"
        if [[ "${name}" == "leaf" ]] ; then
            openssl x509 -req -extfile <(printf "keyUsage=nonRepudiation,digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\nsubjectAltName=DNS:kubenab.kube-system.svc,DNS:localhost") -in "${csrpath}" -CA "${signercert}" -CAkey "${signerkey}" -CAcreateserial -out "${certpath}" -days 365 -sha256
        else
            openssl x509 -req -extfile <(printf "basicConstraints=critical,CA:TRUE,pathlen:${chain_length}\nkeyUsage=critical,cRLSign,digitalSignature,keyCertSign\nextendedKeyUsage=serverAuth\nsubjectAltName=DNS:kubenab.kube-system.svc,DNS:localhost") -in "${csrpath}" -CA "${signercert}" -CAkey "${signerkey}" -CAcreateserial -out "${certpath}" -days 365 -sha256
        fi
        signercert="${certpath}"
        signerkey="${keypath}"
    done
}

main "${@}"