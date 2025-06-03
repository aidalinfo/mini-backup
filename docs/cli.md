# Documentation de la CLI mini-backup

La CLI `backup-cli` permet de gérer vos sauvegardes et restaurations directement depuis la ligne de commande. Voici la liste des commandes disponibles, leurs arguments et des exemples d'utilisation.

---

## Commande principale

```bash
backup-cli <commande> [options]
```

---

## 1. Lister les sauvegardes

### Syntaxe
```bash
backup-cli list backup
```

### Description
Affiche la liste des sauvegardes configurées dans le fichier `config.yaml`.

### Exemple
```bash
docker exec mini-backup /app/backup-cli list backup
```

---

## 2. Restaurer une sauvegarde

### Syntaxe
```bash
backup-cli restore <nom_du_backup> [version]
```

- `<nom_du_backup>` : Nom de la sauvegarde à restaurer (obligatoire).
- `[version]` : Version spécifique à restaurer (optionnel, par défaut : `last`).

### Description
Permet de restaurer une sauvegarde depuis le stockage S3. Si aucune version n'est précisée, la CLI propose de restaurer la dernière version ou de choisir parmi la liste des backups disponibles.

### Exemples
- Restaurer la dernière version d'une sauvegarde :
  ```bash
  docker exec mini-backup /app/backup-cli restore filesystem last
  ```
- Restaurer une version spécifique (sélection interactive) :
  ```bash
  docker exec -it mini-backup /app/backup-cli restore filesystem
  ```

---

## 3. Mettre à jour le logiciel

### Syntaxe
```bash
backup-cli update [--server] [--cli]
```

- `--server` : Met à jour uniquement le serveur.
- `--cli` : Met à jour uniquement la CLI.
- (Sans option) : Met à jour le serveur **et** la CLI.

### Description
Vérifie la dernière version disponible et met à jour les composants sélectionnés.

### Exemples
- Mettre à jour le serveur et la CLI :
  ```bash
  docker exec -it mini-backup /app/backup-cli update
  ```
- Mettre à jour uniquement le serveur :
  ```bash
  docker exec -it mini-backup /app/backup-cli update --server
  ```
- Mettre à jour uniquement la CLI :
  ```bash
  docker exec -it mini-backup /app/backup-cli update --cli
  ```

---

## 4. Ajouter un module

### Syntaxe
```bash
backup-cli modules install <nom_du_module>
```

- `<nom_du_module>` : Nom du module à installer (obligatoire).

### Description
Installe un module officiel ou personnalisé dans mini-backup. Il suffit de spécifier le nom du module, sans options supplémentaires.

### Exemples
- Installer un module officiel (exemple : mysql) :
  ```bash
  docker exec -it mini-backup /app/backup-cli modules install mysql
  ```
- Installer un module personnalisé :
  ```bash
  docker exec -it mini-backup /app/backup-cli modules install custom
  ```

> Pour la liste complète des modules disponibles, utilisez :
> ```bash
> backup-cli modules list
> ```

---

## Notes complémentaires
- Toutes les commandes doivent être exécutées dans le conteneur Docker si vous utilisez l'installation par Docker Compose.
- Pour afficher l'aide générale ou celle d'une commande spécifique :
  ```bash
  backup-cli --help
  backup-cli <commande> --help
  ```
