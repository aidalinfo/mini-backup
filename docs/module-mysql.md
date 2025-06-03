# Module MySQL

Ce module permet de sauvegarder des bases de données MySQL.

## Exemple de configuration YAML
```yaml
backups:
  sqlserver-01:
    type: mysql
    mysql:
      all: true
      host: "localhost"
      port: "3306"
      user: "root"
      password: "example"
      ssl: "false"
    path:
      local: "./backups"
      s3: "backup/glpi-dev/mysql"
    retention:
      standard:
        days: 20
    schedule:
      standard: "*/9 * * * *"
```

## Champs obligatoires
- `type`: Doit être `mysql`
- `mysql.host`, `mysql.port`, `mysql.user`, `mysql.password`
- `path.local`, `path.s3`
- `retention.standard.days`
- `schedule.standard`

## Exemple de commande CLI
```bash
docker exec -it mini-backup /app/backup-cli modules install mysql
```
