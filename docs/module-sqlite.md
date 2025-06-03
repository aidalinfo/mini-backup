# Module SQLite

Ce module permet de sauvegarder des bases de données SQLite.

## Exemple de configuration YAML
```yaml
backups:
  sqlite-new:
    type: sqlite
    sqlite:
      paths:
        - "./datatest/sqlite.db"
    path:
      local: "./backups"
      s3: "minio-data"
    retention:
      standard:
        days: 14
    schedule:
      standard: "*/59 * * * *"
```

## Champs obligatoires
- `type`: Doit être `sqlite`
- `sqlite.paths`: Liste des fichiers à sauvegarder
- `path.local`, `path.s3`
- `retention.standard.days`
- `schedule.standard`

## Exemple de commande CLI
```bash
docker exec -it mini-backup /app/backup-cli modules install sqlite
```
