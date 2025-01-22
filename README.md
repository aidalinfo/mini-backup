# mini-backup 🚀

[![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)](http://forthebadge.com)[![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)](https://forthebadge.com)[![forthebadge](https://forthebadge.com/images/badges/made-with-vue.svg)](https://forthebadge.com)

**mini-backup** est un outil de sauvegarde modulaire écrit en **Go**. L'objectif est de fournir une solution simple, fiable, et performante pour sauvegarder différents types de données, tout en offrant une gestion via une interface utilisateur ainsi q'une **CLI**.

Une documentation est en cours de construction, mais il y a tout de même plus d'information : [documentation](https://mini-backup.aidalinfo.fr/) .

## Fonctionnalités principales

### Types de sauvegarde pris en charge
- **Bases de données** : MySQL/Mariadb et MongoDB.
- **Fichiers/Dossiers locaux** : Sauvegarde des fichiers et répertoires.
- **Stockage S3** : Gestion des objets dans un stockage compatible avec Amazon S3.

### Compression des sauvegardes
- Les sauvegardes sont compressées au format **tar.gz** pour économiser de l’espace et faciliter le transfert.

### Répartition des tâches
- **Rétention automatique** : Les sauvegardes sont conservées pendant une période configurable.
- **Archivage dans Glacier** : Bientôt il sera possible de déposer les sauvegardes dans un stockage froid.

### Configuration flexible
- Utilisation d’un fichier de configuration **YAML** pour définir les comportements de l’outil (sources à sauvegarder, destination, politiques de rétention, etc.).

### CLI externe
- Commandes disponibles pour exécuter et gérer les sauvegardes. (uniquement lister et restaurer pour l'instant)


## License

Ce projet est sous licence ``MIT`` - voir le fichier [LICENSE.md](LICENSE.md) pour plus d'informations