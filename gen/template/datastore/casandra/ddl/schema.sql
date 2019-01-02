DROP TABLE IF EXISTS dummy_type;
CREATE OR REPLACE TABLE dummy_type (
  id       float PRIMARY KEY,
  name     STRING,
  modified TIMESTAMP
);


DROP TABLE IF EXISTS dummy;
CREATE OR REPLACE TABLE dummy (
  id         float PRIMARY KEY,
  type_id    float,
  name       STRING,
  modified   TIMESTAMP
);
