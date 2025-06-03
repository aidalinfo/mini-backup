# Module Filesystem (fs)

Ce module permet de sauvegarder des fichiers ou dossiers locaux.

## Exemple de configuration YAML
```yaml
filesystem:
  type: fs
  fs:
    paths:
      - "/chemin/vers/mon/dossier"
  path:
    local: "./backups/filesystem"
    s3: "minio-data"
  retention:
    standard:
      days: 14
  schedule:
    standard: "*/2 * * * *"
```

## Champs obligatoires
- `type`: Doit être `fs`
- `fs.paths`: Liste des chemins à sauvegarder
- `path.local`: Dossier local de destination
- `retention.standard.days`: Nombre de jours de rétention
- `schedule.standard`: Cron de planification

## Exemple de commande CLI
```bash
docker exec -it mini-backup /app/backup-cli modules install fs
```
