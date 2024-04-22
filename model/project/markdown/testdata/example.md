## Workflow run
### Structure
```text
├── run.yaml
├── blah.json
├── system.yaml
├── database
│   └── database.yaml
└── regression
    └── regression.yaml
```
### Definition
#### run.yaml
```yaml
description: run workflow
init:
    mysqlCred: mysql-e2e.json
    mysqlSecrets: ${secrets.$mysqlCred}
    records: []
    workflowPath: $WorkingDirectory(.)
    CONTACTS: {}
    blah: '@blah'
pipeline:
    init:
        system:
            action: run
            uri: system
        database:
            action: run
            uri: database/database
    test:
        action: run
        uri: regression/regression
    validate:
        action: print
        input:
            message: validation ...
```
### Assets
#### blah.json
```json
{"test": 1}
```
## Workflow system
### Structure
```text
└── system.yaml
```
### Definition
#### system.yaml
```yaml
pipeline:
    stop:
        services:
            service: docker
            action: docker:stop
            input:
                images:
                    - mysql
    services:
        mysql_db1:
            service: docker
            action: docker:run
            sleepTimeMs: 2000
            input:
                env:
                    MYSQL_ROOT_PASSWORD: ${mysqlSecrets.Password}
                image: mysql:5.7
                name: mysql_db1
                platform: linux/amd64
                ports:
                    "3306": !!float 3306
```
## Workflow database
### Structure
```text
└── database
    ├── database.yaml
    └── db1
        └── script
            └── schema.sql
```
### Definition
#### database/database.yaml
```yaml
pipeline:
    register:
        description: register data store db with mysql dsn
        service: dsunit
        action: dsunit:init
        input:
            admin:
                config:
                    credentials: $mysqlCred
                    driver: mysql
                    dsn: '[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true'
                datastore: mysql
                ping: true
            config:
                credentials: $mysqlCred
                driver: mysql
                dsn: '[username]:[password]@tcp(127.0.0.1:3306)/db1?parseTime=true'
            datastore: db1
            scripts:
                - URL: ${workflowPath}/database/db1/script/schema.sql
```
### Assets
#### database/db1/script/schema.sql
```sql


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
```
## Workflow regression
### Structure
```text
└── regression
    ├── regression.yaml
    ├── cases
    │   ├── 001_case1
    │   │   └── prepare
    │   │       ├── CONTACTS.json
    │   │       └── USER_ROLES.json
    │   ├── 002_case2
    │   │   └── prepare
    │   │       ├── CONTACTS.json
    │   │       └── USER_ROLES.json
    │   └── 003_case3
    │       ├── prepare
    │       │   ├── CONTACTS.json
    │       │   └── USER_ROLES.json
    │       └── test.yaml
    ├── default
    │   ├── prepare
    │   │   └── CONTACTS.json
    │   └── test.yaml
    └── database
        └── database.yaml
```
### Definition
#### regression/regression.yaml
```yaml
pipeline:
    database:
        action: run
        uri: regression/database/database
    test:
        subpath: cases/${index}_*
        data:
            '[]dbsetup': '@prepare'
        template:
            step1:
                action: run
                uri: test
```
### Assets
#### regression/cases/001_case1/prepare/CONTACTS.json
```json
[
  {
    "ID": "$Sequences.CONTACTS/${tag}.1",
    "NAME": "Dept",
    "PHONE": "PHONE_2",
    "ENABLED": 1,
    "STR_ID": "STR_ID_${Sequences.CONTACTS}"
  },
  {
    "ID": "$Sequences.CONTACTS/${tag}.2",
    "NAME": "IP",
    "ENABLED": 1,
    "PHONE": "PHONE_2",
    "STR_ID": "STR_ID_${Sequences.CONTACTS}"
  }
]
```
#### regression/cases/001_case1/prepare/USER_ROLES.json
```json
[
  {
    "USER_ID": "$Sequences.CONTACTS/${tag}.1",
    "AUTHORITY": "ROLE_BUSINESS_OWNER",
    "CREATED_USER": 10
  },
  {
    "USER_ID": "$Sequences.CONTACTS/${tag}.2",
    "AUTHORITY": "ROLE_BUSINESS_OWNER",
    "CREATED_USER": 11
  }
]
```
#### regression/cases/002_case2/prepare/CONTACTS.json
```json
[
  {
    "ID": "$Sequences.CONTACTS/${tag}.1",
    "NAME": "Dept",
    "PHONE": "PHONE_2",
    "ENABLED": 1,
    "STR_ID": "STR_ID_${Sequences.CONTACTS}"
  },
  {
    "ID": "$Sequences.CONTACTS/${tag}.2",
    "NAME": "IP",
    "ENABLED": 1,
    "PHONE": "PHONE_2",
    "STR_ID": "STR_ID_${Sequences.CONTACTS}"
  }
]
```
#### regression/cases/002_case2/prepare/USER_ROLES.json
```json
[
  {
    "USER_ID": "$Sequences.CONTACTS/${tag}.1",
    "AUTHORITY": "ROLE_BUSINESS_OWNER",
    "CREATED_USER": 10
  },
  {
    "USER_ID": "$Sequences.CONTACTS/${tag}.2",
    "AUTHORITY": "ROLE_BUSINESS_OWNER",
    "CREATED_USER": 11
  }
]
```
#### regression/cases/003_case3/prepare/CONTACTS.json
```json
[
  {
    "ID": "$Sequences.CONTACTS/${tag}.1",
    "NAME": "Dept",
    "PHONE": "PHONE_2",
    "ENABLED": 1,
    "STR_ID": "STR_ID_${Sequences.CONTACTS}"
  },
  {
    "ID": "$Sequences.CONTACTS/${tag}.2",
    "NAME": "IP",
    "ENABLED": 1,
    "PHONE": "PHONE_2",
    "STR_ID": "STR_ID_${Sequences.CONTACTS}"
  }
]
```
#### regression/cases/003_case3/prepare/USER_ROLES.json
```json
[
  {
    "USER_ID": "$Sequences.CONTACTS/${tag}.1",
    "AUTHORITY": "ROLE_BUSINESS_OWNER",
    "CREATED_USER": 10
  },
  {
    "USER_ID": "$Sequences.CONTACTS/${tag}.2",
    "AUTHORITY": "ROLE_BUSINESS_OWNER",
    "CREATED_USER": 11
  }
]
```
#### regression/default/prepare/CONTACTS.json
```json
[]
```
#### regression/default/test.yaml
```yaml
init:
  setup: ${CONTACTS.${tag}.1}
pipeline:
  test:
    action: print
    message: '$tag: setup data  $AsJSON(${setup})'
```
## Workflow database
### Structure
```text
└── regression
    └── database
        └── database.yaml
```
### Definition
#### regression/database/database.yaml
```yaml
pipeline:
    loadSequences:
        description: task returns values of the sequence for supplied tables
        service: dsunit
        action: dsunit:sequence
        input:
            datastore: db1
            tables: $StringKeys(${data.dbsetup})
        post:
            - name: Sequences
              value: $Sequences
    printSequences:
        action: print
        input:
            message: $AsJSON($Sequences)
    allocateSequences:
        action: nop
        input: {}
    recordInfo:
        action: print
        input:
            message: $AsJSON($records)
    populate:
        when: $Len($records) > 0
        service: dsunit
        action: dsunit:prepare
        input:
            data: $records
            datastore: db1
```
## Workflow test
### Structure
```text
└── regression
    └── cases
        └── 003_case3
            ├── test.yaml
            └── prepare
                ├── CONTACTS.json
                └── USER_ROLES.json
```
### Definition
#### regression/cases/003_case3/test.yaml
```yaml
init:
    setup: ${CONTACTS.${tag}.1}
pipeline:
    test:
        action: print
        input:
            message: '$tag: setup data  $AsJSON(${setup})'
```
## Workflow database
### Structure
```text
└── regression
    └── database
        └── database.yaml
```
### Definition
#### regression/database/database.yaml
```yaml
pipeline:
    loadSequences:
        description: task returns values of the sequence for supplied tables
        service: dsunit
        action: dsunit:dsunit:sequence
        input:
            tables: $StringKeys(${data.dbsetup})
            datastore: db1
        post:
            - name: Sequences
              value: $Sequences
    printSequences:
        action: print
        input:
            message: $AsJSON($Sequences)
    allocateSequences:
        action: nop
        input: {}
    recordInfo:
        action: print
        input:
            message: $AsJSON($records)
    populate:
        when: $Len($records) > 0
        service: dsunit
        action: dsunit:dsunit:prepare
        input:
            data: $records
            datastore: db1
```
## Workflow test
### Structure
```text
└── regression
    └── cases
        └── 003_case3
            ├── test.yaml
            └── prepare
                ├── CONTACTS.json
                └── USER_ROLES.json
```
### Definition
#### regression/cases/003_case3/test.yaml
```yaml
init:
  setup: ${CONTACTS.${tag}.1}
pipeline:
  test:
    action: print
    input:
      message: '$tag: setup data  $AsJSON(${setup})'
```