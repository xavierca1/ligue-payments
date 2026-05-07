#!/bin/bash

# Script para testar DocuSeal carregando variáveis do .env

set -a
source .env
set +a

go test ./internal/usecase -run TestDocuSealAutomaticSubmission -v
