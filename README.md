# mini-backup 🚀

[![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)](http://forthebadge.com)[![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)](https://forthebadge.com)[![forthebadge](https://forthebadge.com/images/badges/made-with-vue.svg)](https://forthebadge.com)

**mini-backup** est un outil de sauvegarde modulaire écrit en **Go**, conçu pour répondre aux besoins des infrastructures modernes, qu’elles soient **cloud**, **multicloud**, ou locales. Il offre une gestion simplifiée et sécurisée des sauvegardes grâce à une configuration basée sur **YAML**, une **CLI** intuitive, et une interface web.

Découvrez la documentation complète ici : [Documentation officielle](https://mini-backup.aidalinfo.fr/).
---

## Fonctionnalités principales

### Types de sauvegardes prises en charge
- **Bases de données** : MySQL/MariaDB et MongoDB.
- **Fichiers/Dossiers locaux** : Sauvegardez des fichiers et répertoires avec compression et chiffrement.
- **Stockage S3** : Gérez vos sauvegardes dans des solutions compatibles S3.

### Compression et chiffrement
- Les sauvegardes sont compressées au format **tar.gz** pour optimiser l’espace de stockage.
- Les données sont chiffrées avec **AES-256** avant d’être envoyées vers S3.

### Gestion des sauvegardes
- **Rétention configurable** :
  - Standard : Durée de conservation dans le stockage principal S3.
  - Glacier : Archivage des sauvegardes pour une conservation à long terme (fonctionnalité en cours de développement).
- **Planification automatisée** : Expressions CRON pour définir les horaires de sauvegarde.
- **Restauration facile** : Via l’interface web ou la CLI.

### Interface web et CLI
- **Interface web** :
  - Visualisez vos configurations de sauvegarde.
  - Lancez des restaurations en quelques clics.
- **CLI** :
  - Listez et restaurez vos sauvegardes depuis la ligne de commande.

### Configuration flexible
- La configuration repose sur deux fichiers principaux :
  - **config.yaml** : Définit les tâches de sauvegarde (sources, destinations, rétention, planification).
  - **server.yaml** : Configure les endpoints S3 et les paramètres du serveur.

---

## Prérequis

- **Docker** et **Docker Compose** pour une installation simple.
- Une clé **AES-256** pour le chiffrement (peut être générée avec la commande suivante) :
  ```bash
  openssl rand -base64 32
  ```

---

## Installation rapide

1. Téléchargez le projet avec **wget** ou **curl** :
   ```bash
   wget https://github.com/aidalinfo/mini-backup-getting-started/archive/refs/heads/main.zip -O mini-backup-getting-started.zip
   ```
   ```bash
   curl -L https://github.com/aidalinfo/mini-backup-getting-started/archive/refs/heads/main.zip -o mini-backup-getting-started.zip
   ```

2. Décompressez l’archive et accédez au dossier :
   ```bash
   unzip mini-backup-getting-started.zip
   cd mini-backup-getting-started-main
   ```

3. Lancez les conteneurs avec Docker Compose :
   ```bash
   docker compose up -d
   ```

4. Modifiez les fichiers de configuration générés (`config.yaml` et `server.yaml`) selon vos besoins.

---

## Modifier server.yaml

Éditez le fichier `server.yaml` pour indiquer le nom du bucket S3 et les identifiants de connexion pour héberger vos sauvegardes.

Voici un exemple de la section `rstorage` après modification :

```yaml
rstorage:
  minio:
    endpoint: "http://minio:9000"
    bucket_name: "backup"
    access_key: "${{MINIO_ACCESS_KEY}}"
    secret_key: "${{MINIO_SECRET_KEY}}"
    region: "fr-par"
```

---

## Restauration

### Interface web
- Accédez à [http://localhost](http://localhost).
- Sélectionnez une sauvegarde et lancez la restauration en un clic.

### CLI
- Restaurer le dernier backup :
  ```bash
  docker exec mini-backup /app/backup-cli restore <nom_du_backup> last
  ```
- Restaurer un backup spécifique :
  ```bash
  docker exec -it mini-backup /app/backup-cli restore <nom_du_backup>
  ```

---

## Roadmap

Voici un aperçu des fonctionnalités à venir :
- Sauvegarde de bases PostgreSQL.
- Gestion des sauvegardes Kubernetes.
- Archivage avancé avec intégration Glacier.
- Documentation détaillée pour chaque type de sauvegarde.
- Gestion des secrets avec un gestionnaire tel qu’Infisical.

---

## Contribuer

Nous accueillons les contributions ! Si vous souhaitez participer, n’hésitez pas à ouvrir une **issue** ou une **pull request** sur le [dépôt GitHub](https://github.com/aidalinfo/mini-backup).

---

## Licence

Ce projet est sous licence **MIT**. Vous êtes libre de l'utiliser et de le modifier conformément aux termes de cette licence.
