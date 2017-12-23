DROP TABLE IF EXISTS apps;

CREATE TABLE apps (
  id   INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255),
  url  VARCHAR(255)
);

DROP TABLE IF EXISTS users;
CREATE TABLE users (
  id    INT AUTO_INCREMENT PRIMARY KEY,
  email VARCHAR(255)
);

DROP TABLE IF EXISTS app_usage;
CREATE TABLE app_usage (
  id      INT AUTO_INCREMENT PRIMARY KEY,
  date    DATE,
  count   INT,
  user_id INT,
  app_id  INT
);


GRANT ALL ON *.* TO 'dev'@'%' IDENTIFIED BY 'dev';
