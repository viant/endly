DROP TABLE IF EXISTS ASSET;
DROP TABLE IF EXISTS TASK;
DROP TABLE IF EXISTS WORKFLOW;
DROP TABLE IF EXISTS PROJECT;

-- Project Table
CREATE TABLE PROJECT (
                         ID TEXT PRIMARY KEY,
                         NAME TEXT NOT NULL,
                         DESCRIPTION TEXT
);

-- Workflow Table
CREATE TABLE WORKFLOW (
                          ID TEXT PRIMARY KEY,
                          REVISION TEXT,
                          URI TEXT,
                          PROJECT_ID TEXT,
                          NAME TEXT,
                          INIT BLOB, -- JSON Array
                          POST BLOB, -- JSON Array
                          TEMPLATE TEXT,
                          INSTANCE_INDEX INTEGER,
                          INSTANCE_TAG TEXT,
                          FOREIGN KEY(PROJECT_ID) REFERENCES PROJECT(ID)
);




-- Task Table
CREATE TABLE TASK (
                      ID TEXT PRIMARY KEY,
                      PARENT_ID TEXT,
                      POSITION INTEGER,
                      TAG TEXT,
                      INIT BLOB, -- JSON Array
                      POST BLOB, -- JSON Array
                      DESCRIPTION TEXT,
                      WHEN_EXPR TEXT,
                      EXIT_EXPR TEXT,
                      ON_ERROR TEXT,
                      DEFERRED TEXT,
                      SERVICE TEXT,
                      ACTION TEXT,
                      INPUT TEXT,
                      INPUT_URI TEXT,
                      SKIP_EXPR TEXT,
                      ASYNC BOOLEAN,
                      FAIL BOOLEAN,
                      IS_TEMPLATE BOOLEAN,
                      SUB_PATH TEXT,
                      RANGE_EXPR TEXT,
                      DATA BLOB, -- JSON Object
                      VARIABLES BLOB, -- JSON Array
                      EXTRACTS BLOB, -- JSON Object
                      SLEEP_TIME_MS INTEGER,
                      THINK_TIME_MS INTEGER,
                      LOGGING BOOLEAN,
                      REPEAT_RUN INTEGER
);

-- Asset Table
CREATE TABLE ASSET (
                       ID TEXT PRIMARY KEY,
                       LOCATION TEXT,
                       DESCRIPTION TEXT,
                       WORKFLOW_ID TEXT,
                       IS_DIR BOOLEAN,
                       TEMPLATE TEXT,
                       INSTANCE_INDEX INTEGER,
                       INSTANCE_TAG TEXT,
                       POSITION INTEGER,
                       SOURCE BLOB,
                       FORMAT TEXT,
                       CODEC TEXT,
                       FOREIGN KEY(WORKFLOW_ID) REFERENCES WORKFLOW(ID)
);