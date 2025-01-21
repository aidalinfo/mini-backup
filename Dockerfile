# Étape 1 : Construction du programme Go
FROM golang:1.23.4 AS builder

# Définir le répertoire de travail
WORKDIR /app

# Copier les fichiers Go et les dépendances
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compiler un binaire statiquement lié
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o backup-server cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o backup-cli cli/main.go

# Étape 2 : Image minimale pour exécuter le binaire et les outils
FROM alpine:latest

# Installer les outils nécessaires
RUN apk add --no-cache \
  mongodb-tools \
  mysql-client \
  bash

# Définir le répertoire de travail
WORKDIR /app

# Copier le binaire Go depuis l'étape de build
COPY --from=builder /app/backup-server .
COPY --from=builder /app/backup-cli .

# Vérifier les permissions (non nécessaire si le binaire est bien construit)
RUN chmod +x backup-server && chmod +x backup-cli

EXPOSE 8080

# Commande par défaut
CMD ["/app/backup-server"]
