DROP TABLE IF EXISTS dummy_type;

CREATE TABLE dummy_type (
  id       INT AUTO_INCREMENT PRIMARY KEY,
  name     VARCHAR(255) DEFAULT NULL,
  modified TIMESTAMP    DEFAULT current_timestamp,
  UNIQUE(name)
);


DROP TABLE IF EXISTS dummy;

CREATE TABLE dummy (
  id         INT AUTO_INCREMENT PRIMARY KEY,
  type_id   INT NOT NULL,
  name       VARCHAR(255) DEFAULT NULL,
  modified   TIMESTAMP    DEFAULT current_timestamp
);


