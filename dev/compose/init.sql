-- Créer la base de données
CREATE DATABASE IF NOT EXISTS glpidb;
USE glpidb;

-- Créer la table restaurants
CREATE TABLE IF NOT EXISTS restaurants (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255),
    city VARCHAR(100),
    country VARCHAR(100),
    phone VARCHAR(20),
    email VARCHAR(255),
    cuisine_type VARCHAR(100),
    rating DECIMAL(3, 1),
    opening_hours VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insérer des données de test
INSERT INTO restaurants (name, address, city, country, phone, email, cuisine_type, rating, opening_hours) VALUES
('Le Bistrot', '123 Rue de Paris', 'Paris', 'France', '0123456789', 'bistrot@example.com', 'French', 4.5, '10:00-22:00'),
('Pizza Palace', '456 Via Roma', 'Rome', 'Italy', '0987654321', 'pizza@example.com', 'Italian', 4.2, '11:00-23:00'),
('Sushi Heaven', '789 Tokyo St', 'Tokyo', 'Japan', '1234567890', 'sushi@example.com', 'Japanese', 4.7, '12:00-22:00'),
('Burger Joint', '101 Burger Ave', 'New York', 'USA', '5551234567', 'burger@example.com', 'American', 4.0, '11:00-21:00'),
('Taco Fiesta', '202 Taco Blvd', 'Mexico City', 'Mexico', '5559876543', 'taco@example.com', 'Mexican', 4.4, '10:00-22:00'),
('Curry House', '303 Spice Rd', 'London', 'UK', '02071231234', 'curry@example.com', 'Indian', 4.6, '11:00-22:00'),
('Seafood Delight', '404 Ocean Dr', 'Sydney', 'Australia', '0291234567', 'seafood@example.com', 'Seafood', 4.8, '12:00-22:00'),
('Vegan Vibes', '505 Green St', 'Berlin', 'Germany', '03012345678', 'vegan@example.com', 'Vegan', 4.3, '11:00-21:00'),
('Steakhouse', '606 Meat Ave', 'Madrid', 'Spain', '911234567', 'steak@example.com', 'Steakhouse', 4.9, '18:00-23:00'),
('Café Central', '707 Coffee Rd', 'Vienna', 'Austria', '01512345678', 'cafe@example.com', 'Café', 4.1, '08:00-20:00');

-- Ajoutez plus d'entrées selon vos besoins
-- Créer la base de données

CREATE DATABASE IF NOT EXISTS bookstore;
USE bookstore;

-- Créer la table books
CREATE TABLE IF NOT EXISTS books (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255),
    genre VARCHAR(100),
    isbn VARCHAR(13),
    price DECIMAL(5, 2),
    stock INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insérer des données de test
INSERT INTO books (title, author, genre, isbn, price, stock) VALUES
('Le Petit Prince', 'Antoine de Saint-Exupéry', 'Fiction', '9782070612703', 12.99, 50),
('1984', 'George Orwell', 'Dystopian', '9780451524935', 9.99, 30),
('To Kill a Mockingbird', 'Harper Lee', 'Fiction', '9780061120084', 8.99, 40),
('The Great Gatsby', 'F. Scott Fitzgerald', 'Classic', '9780743273565', 7.99, 25),
('Pride and Prejudice', 'Jane Austen', 'Romance', '9780199832449', 6.99, 35),
('The Hobbit', 'J.R.R. Tolkien', 'Fantasy', '9780618002214', 10.99, 20),
('The Catcher in the Rye', 'J.D. Salinger', 'Fiction', '9780316769488', 8.99, 30),
('Moby-Dick', 'Herman Melville', 'Adventure', '9780199832821', 11.99, 15),
('War and Peace', 'Leo Tolstoy', 'Historical Fiction', '9780199832791', 14.99, 10),
('The Odyssey', 'Homer', 'Epic', '9780199832760', 9.99, 20);

-- Ajoutez plus d'entrées selon vos besoins
