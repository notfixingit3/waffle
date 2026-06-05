#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL database restore script for Project Syrup
# Supports plain .sql and .sql.gz backup files

DB_CONTAINER="${DB_CONTAINER:-postgres}"
DB_USER="${DB_USER:-syrup}"
DB_NAME="${DB_NAME:-syrup}"

usage() {
    echo "Usage: $0 <backup-file.sql|backup-file.sql.gz>"
    echo ""
    echo "Environment variables:"
    echo "  DB_CONTAINER  Docker container name (default: postgres)"
    echo "  DB_USER       PostgreSQL user (default: syrup)"
    echo "  DB_NAME       PostgreSQL database name (default: syrup)"
    exit 1
}

# Check argument count
if [ "$#" -ne 1 ]; then
    echo "Error: Exactly one argument required (backup file path)"
    usage
fi

BACKUP_FILE="$1"

# Validate file exists
if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: File not found: $BACKUP_FILE"
    exit 1
fi

# Validate file is non-empty
if [ ! -s "$BACKUP_FILE" ]; then
    echo "Error: File is empty: $BACKUP_FILE"
    exit 1
fi

# Validate file extension
if [[ "$BACKUP_FILE" != *.sql && "$BACKUP_FILE" != *.sql.gz ]]; then
    echo "Error: File must end in .sql or .sql.gz"
    exit 1
fi

# Prompt for confirmation
echo "This will overwrite database '$DB_NAME'. Continue? [y/N]"
read -r response

if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo "Restore cancelled by user."
    exit 0
fi

# Perform restore
echo "Restoring database '$DB_NAME' from '$BACKUP_FILE'..."

if [[ "$BACKUP_FILE" == *.sql.gz ]]; then
    gunzip < "$BACKUP_FILE" | docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" "$DB_NAME"
else
    docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" "$DB_NAME" < "$BACKUP_FILE"
fi

echo "Successfully restored database '$DB_NAME' from '$BACKUP_FILE'"
