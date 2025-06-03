# Module MongoDB

Ce module permet de sauvegarder des bases de données MongoDB.

## Exemple de configuration YAML
```yaml
backups:
  mongo:
    type: mongo
    mongo:
      host: "localhost"
      port: "27017"
      user: "root"
      password: "example"
      ssl: false
    path:
      local: "./backups"
      s3: "backup/mongo/mongo"
    retention:
      standard:
        days: 14
    schedule:
      standard: "*/2 * * * *"
```

## Champs obligatoires
- `type`: Doit être `mongo`
- `mongo.host`, `mongo.port`, `mongo.user`, `mongo.password`
- `path.local`, `path.s3`
- `retention.standard.days`
- `schedule.standard`

## Exemple de commande CLI
```bash
docker exec -it mini-backup /app/backup-cli modules install mongo
```
