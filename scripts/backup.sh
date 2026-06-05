#!/usr/bin/env bash
set -euo pipefail

# Project Syrup Database Backup Script
# Standalone backup using docker exec (works on any Docker host).
# Usage:
#   ./scripts/backup.sh
#   DB_CONTAINER=mydb DB_USER=admin DB_NAME=mydb BACKUP_DIR=/backups ./scripts/backup.sh

# ─── Configuration ────────────────────────────────────────────────────────────
DB_CONTAINER="${DB_CONTAINER:-postgres}"
DB_USER="${DB_USER:-syrup}"
DB_NAME="${DB_NAME:-syrup}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"

# ─── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ─── Helpers ──────────────────────────────────────────────────────────────────
error_exit() {
    echo -e "${RED}ERROR${NC}: $1" >&2
    exit 1
}

success() {
    echo -e "${GREEN}SUCCESS${NC}: $1"
}

info() {
    echo -e "${YELLOW}INFO${NC}: $1"
}

# ─── Preflight Checks ─────────────────────────────────────────────────────────
# Check docker is available
if ! command -v docker &>/dev/null; then
    error_exit "docker command not found. Is Docker installed?"
fi

# Check container is running
if ! docker ps --format '{{.Names}}' | grep -qx "${DB_CONTAINER}"; then
    error_exit "Container '${DB_CONTAINER}' is not running. Start it first."
fi

# ─── Prepare Backup Directory ─────────────────────────────────────────────────
mkdir -p "${BACKUP_DIR}"

# ─── Generate Filename ────────────────────────────────────────────────────────
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
FILENAME="syrup-backup-${TIMESTAMP}.sql.gz"
BACKUP_PATH="${BACKUP_DIR}/${FILENAME}"

info "Starting backup of database '${DB_NAME}' from container '${DB_CONTAINER}'..."
info "Backup will be written to: ${BACKUP_PATH}"

# ─── Run Backup ───────────────────────────────────────────────────────────────
# Use a temporary file to avoid creating empty files on failure
TMP_FILE=$(mktemp "${BACKUP_DIR}/.tmp-backup-XXXXXX.sql.gz")

cleanup_tmp() {
    if [[ -f "${TMP_FILE}" ]]; then
        rm -f "${TMP_FILE}"
    fi
}
trap cleanup_tmp EXIT

if ! docker exec "${DB_CONTAINER}" pg_dump -U "${DB_USER}" "${DB_NAME}" | gzip > "${TMP_FILE}"; then
    error_exit "pg_dump failed. Check PostgreSQL logs and credentials."
fi

# ─── Verify Output ────────────────────────────────────────────────────────────
if [[ ! -f "${TMP_FILE}" ]]; then
    error_exit "Backup file was not created."
fi

FILE_SIZE=$(stat -f%z "${TMP_FILE}" 2>/dev/null || stat -c%s "${TMP_FILE}" 2>/dev/null || echo "0")
if [[ "${FILE_SIZE}" -eq 0 ]]; then
    error_exit "Backup file is empty (0 bytes)."
fi

# Move to final destination
mv "${TMP_FILE}" "${BACKUP_PATH}"

# ─── Success ──────────────────────────────────────────────────────────────────
success "Backup completed: ${BACKUP_PATH}"
info "File size: ${FILE_SIZE} bytes"
