#!/bin/bash

# Vérification de la disponibilité de curl ou wget
if command -v curl >/dev/null 2>&1; then
    DOWNLOAD_CMD="curl -sSL -o"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD_CMD="wget -qO"
else
    echo "Erreur : ni curl ni wget n'est installé."
    exit 1
fi

# URL de base du dépôt
REPO_URL="https://github.com/aidalinfo/mini-backup"
RAW_BASE_URL="https://raw.githubusercontent.com/aidalinfo/mini-backup/main"
RELEASES_BASE_URL="https://github.com/aidalinfo/mini-backup/releases/download"

# Récupération de la dernière version
echo "Récupération de la dernière version..."
LATEST_VERSION=$(curl -sSL "https://api.github.com/repos/aidalinfo/mini-backup/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST_VERSION" ]; then
    echo "Erreur : Impossible de récupérer la dernière version."
    exit 1
fi
echo "Dernière version détectée : $LATEST_VERSION"

# Répertoire temporaire pour le téléchargement
TMP_DIR=$(mktemp -d)
echo "Répertoire temporaire : $TMP_DIR"

# Téléchargement des fichiers
echo "Téléchargement des fichiers nécessaires..."
$DOWNLOAD_CMD "$TMP_DIR/.env" "$RAW_BASE_URL/examples/.env.example"
$DOWNLOAD_CMD "$TMP_DIR/config.yaml" "$RAW_BASE_URL/examples/config.yaml"
$DOWNLOAD_CMD "$TMP_DIR/server.yaml" "$RAW_BASE_URL/examples/server.yaml"
$DOWNLOAD_CMD "$TMP_DIR/mini-backup-cli" "$RELEASES_BASE_URL/$LATEST_VERSION/mini-backup-cli_linux_arm64"
$DOWNLOAD_CMD "$TMP_DIR/mini-backup" "$RELEASES_BASE_URL/$LATEST_VERSION/mini-backup_linux_amd64"

# Vérification des téléchargements
if [ $? -ne 0 ]; then
    echo "Erreur lors du téléchargement des fichiers."
    exit 1
fi

# Création des répertoires d'installation
INSTALL_DIR="/etc/mini-backup"
CONFIG_DIR="$INSTALL_DIR/config"
mkdir -p "$CONFIG_DIR"

# Copie des fichiers téléchargés dans leurs emplacements finaux
echo "Copie des fichiers dans leurs emplacements finaux..."
cp "$TMP_DIR/.env" "$INSTALL_DIR/"
cp "$TMP_DIR/config.yaml" "$CONFIG_DIR/"
cp "$TMP_DIR/server.yaml" "$CONFIG_DIR/"
cp "$TMP_DIR/mini-backup-cli" "$INSTALL_DIR/"
cp "$TMP_DIR/mini-backup" "$INSTALL_DIR/"

# Donner les permissions d'exécution aux binaires
chmod +x "$INSTALL_DIR/mini-backup-cli" "$INSTALL_DIR/mini-backup"

# Configuration du service systemd
SERVICE_FILE="/etc/systemd/system/mini-backup.service"
cat << EOF > "$SERVICE_FILE"
[Unit]
Description=Mini backup service
After=network.target

[Service]
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/mini-backup
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
EOF

echo "Fichier de service systemd créé : $SERVICE_FILE"

# Recharger systemd pour prendre en compte le nouveau service
echo "Rechargement de systemd..."
systemctl daemon-reload

# Activer et démarrer le service
echo "Activation et démarrage du service mini-backup..."
systemctl enable mini-backup
systemctl start mini-backup

# Nettoyage du répertoire temporaire
rm -rf "$TMP_DIR"
echo "Installation terminée avec succès."
