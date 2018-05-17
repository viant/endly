CREATE OR REPLACE TABLE dummy_type (
  id       INT64 NOT NULL,
  name     STRING,
  modified TIMESTAMP
);


CREATE OR REPLACE TABLE dummy (
  id         INT64 NOT NULL,
  type_id    INT64 NOT NULL,
  name       STRING,
  modified   TIMESTAMP
);


