DROP TABLE IF EXISTS expenditure ;

CREATE TABLE expenditure (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  country     VARCHAR(255),
  year        int,
  category    VARCHAR(255),
  sub_category VARCHAR(255),
  expenditure DECIMAL(7,2)
);



DROP TABLE IF EXISTS report;

CREATE TABLE report (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255),
  type VARCHAR(255),
  report text
);




DROP TABLE IF EXISTS report_performance;

CREATE TABLE report_performance (
  id          INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255),
  type VARCHAR(255),
  query text,
  time_taken_ms INT,
  run_timestamp TIMESTAMP
);



