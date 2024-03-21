-- Project Table
CREATE TABLE PROJECT (
                         ID TEXT PRIMARY KEY,
                         NAME TEXT
);

-- Workflow Table
CREATE TABLE WORKFLOW (
                          ID TEXT PRIMARY KEY,
                          URI TEXT,
                          REVISION TEXT,
                          PROJECT_ID TEXT,
                          NAME TEXT,
                          INIT BLOB, -- Serialized Variables JSON
                          POST BLOB, -- Serialized Variables JSON
                          TEMPLATE TEXT,
                          FOREIGN KEY(PROJECT_ID) REFERENCES PROJECT(ID)
);

-- Task Table
CREATE TABLE TASK (
                      ID TEXT PRIMARY KEY,
                      PARENT_ID TEXT,
                      IDX INTEGER,
                      TAG TEXT,
                      TAG_INDEX INT,
                      INIT BLOB, -- Serialized Variables JSON
                      POST BLOB, -- Serialized Variables JSON
                      DESCRIPTION TEXT,
                      WHEN_EXPR TEXT,
                      EXIT TEXT,
                      ON_ERROR TEXT,
                      DEFERRED TEXT,
                      SERVICE TEXT,
                      ACTION TEXT,
                      REQUEST TEXT,
                      REQUEST_URI TEXT,
                      ASYNC INTEGER, -- SQLite does not have a boolean type, using INTEGER as a boolean (0 = false, 1 = true)
                      IS_TEMPLATE INTEGER,
                      SKIP TEXT,
                      SUB_PATH TEXT,
                      RANGE TEXT,
                      DATA BLOB, -- Serialized map[string]string JSON
                      VARIABLES BLOB, -- Serialized Variables JSON
                      EXTRACTS BLOB, -- Serialized Variables JSON
                      SLEEP_TIME_MS INTEGER,
                      THINK_TIME_MS INTEGER,
                      LOGGING INTEGER, -- As with ASYNC, using INTEGER as boolean
                      REPEAT INTEGER,
                      WORKFLOW_ID TEXT,
                      FOREIGN KEY(WORKFLOW_ID) REFERENCES WORKFLOW(ID)
);

-- Revision Table
CREATE TABLE REVISION (
                          ID TEXT PRIMARY KEY,
                          USER TEXT,
                          COMMENT TEXT,
                          DIFF TEXT
);


-- Asset Table
CREATE TABLE ASSET (
                       ID TEXT PRIMARY KEY,
                       LOCATION TEXT,
                       DESCRIPTION TEXT,
                       WORKFLOW_ID TEXT,
                       TAG_ID TEXT,
                       IDX INTEGER,
                       SOURCE TEXT,
                       FORMAT TEXT,
                       CODEC TEXT,
                       FOREIGN KEY(WORKFLOW_ID) REFERENCES WORKFLOW(ID)
);