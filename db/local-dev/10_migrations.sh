#!/bin/sh
# Прогоняет db/migration/V*.sql по порядку номеров, как deploy/deploy.sh.
set -e
for f in $(ls /migrations/V*__*.sql | sort -V); do
  echo "applying $f"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$f"
done
