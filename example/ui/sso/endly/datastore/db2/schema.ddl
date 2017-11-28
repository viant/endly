DROP TABLE IF EXISTS user ;

CREATE TABLE user (
  id            INT AUTO_INCREMENT PRIMARY KEY,
  email         VARCHAR(255) NOT NULL,
  name          VARCHAR(255) NOT NULL,
  encrypted_password VARCHAR(255) NOT NULL,
  data_of_birth DATE,
  removed       INT,
  last_singin   TIMESTAMP,
  sigin_count   int,
  CONSTRAINT user_unique_email UNIQUE (email)
);
