# Module GitHub

Ce module permet de sauvegarder des dépôts GitHub.

## Exemple de configuration YAML
```yaml
backups:
  github:
    type: github
    github:
      token: "example"
      org:
        - killian
        - aidalinfo
    path:
      local: "./backups"
      s3: "backup/github"
    retention:
      standard:
        days: 14
    schedule:
      standard: "*/2 * * * *"
```

## Champs obligatoires
- `type`: Doit être `github`
- `github.token`, `github.org`: Token d’accès et liste d’organisations
- `path.local`, `path.s3`
- `retention.standard.days`
- `schedule.standard`

## Exemple de commande CLI
```bash
docker exec -it mini-backup /app/backup-cli modules install github
```
