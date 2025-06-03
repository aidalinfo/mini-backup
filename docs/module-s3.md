# Module S3

Ce module permet de sauvegarder des données sur un stockage compatible S3 (ex: MinIO, AWS S3).

## Exemple de configuration YAML
```yaml
backups:
  minio-data:
    type: s3
    s3:
      all: true
      endpoint: "http://localhost:9002"
      region: "fr-par"
      ACCESS_KEY: "minioadmin"
      SECRET_KEY: "miniopassword"
      pathStyle: true
    path:
      local: "./backups"
      s3: "backup/minio-data"
    retention:
      standard:
        days: 14
    schedule:
      standard: "*/2 * * * *"
```

## Champs obligatoires
- `type`: Doit être `s3`
- `s3.endpoint`, `s3.region`, `s3.ACCESS_KEY`, `s3.SECRET_KEY`
- `path.local`, `path.s3`
- `retention.standard.days`
- `schedule.standard`

## Exemple de commande CLI
```bash
docker exec -it mini-backup /app/backup-cli modules install s3
```
