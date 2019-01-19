CREATE DATABASE `observer`;

USE `observer`;

DROP TABLE IF EXISTS items;

CREATE TABLE items (
		id int(11) NOT NULL AUTO_INCREMENT,
  	title varchar(255) NOT NULL,
  	description text NOT NULL,
  	updated varchar(255) DEFAULT NULL,
  	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO items (id, title, description, updated) VALUES
	(1,	'title 1', 'description 1',	'vantonyuk'),
	(2,	'title 2', 'description 2',	NULL);

DROP TABLE IF EXISTS users;

CREATE TABLE users (
	user_id int(11) NOT NULL AUTO_INCREMENT,
		login varchar(255) NOT NULL,
		password varchar(255) NOT NULL,
		email varchar(255) NOT NULL,
		info text NOT NULL,
		updated varchar(255) DEFAULT NULL,
		PRIMARY KEY (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO users (user_id, login, password, email, info, updated) VALUES
	(1,	'vantonyuk', '12345', 'vantonyuk@example.com', 'info',	NULL);