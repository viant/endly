

-- Drop the USER_ROLES table if it exists
DROP TABLE IF EXISTS USER_ROLES;

-- Drop the CONTACTS table if it exists
DROP TABLE IF EXISTS CONTACTS;

-- Create the CONTACTS table
CREATE TABLE CONTACTS (
                          ID int(11) AUTO_INCREMENT NOT NULL ,
                          NAME VARCHAR(255),
                          PHONE VARCHAR(255),
                          ENABLED TINYINT(1),
                          STR_ID VARCHAR(255),
                          PRIMARY KEY (ID)
);

-- Create the USER_ROLES table
CREATE TABLE USER_ROLES (
                            USER_ID int(11),
                            AUTHORITY VARCHAR(255),
                            CREATED_USER INT,
                            FOREIGN KEY (USER_ID) REFERENCES CONTACTS(ID)
);