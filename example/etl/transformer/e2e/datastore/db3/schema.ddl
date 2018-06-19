DROP TABLE IF EXISTS apps;

CREATE TABLE apps (
  id   INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255),
  url  VARCHAR(255)
);

DROP TABLE IF EXISTS users;
CREATE TABLE users (
  id    INT AUTO_INCREMENT PRIMARY KEY,
  email VARCHAR(255),
  dob  DATE
);

DROP TABLE IF EXISTS user_visits;
CREATE TABLE user_visits (
  visit_date    DATE,
  visit_count   INT,
  visit_app_id  INT,
  user_id INT,
  PRIMARY KEY(user_id, visit_app_id, visit_date)
);


GRANT ALL ON *.* TO 'dev'@'%' IDENTIFIED BY 'dev';
