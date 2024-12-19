# Migrator

Migrator is a utility that makes working with migrations easier. It adds basic operations such as create table, migrate, rollback, rollback all. Supports only postgres.

## Install
```curl -fsSL https://raw.githubusercontent.com/Dorero/migrator/main/install.sh | bash```


## From source code
```
git clone https://github.com/Dorero/migrator
cd migrator
go build
``` 

## Usage
```migrator init``` - Creates a config for connecting to the database and a migration scheme.

```migrator create <table name>``` - Creates a migration.

```migrator migrate``` - Applies all unapplied migrations.

```migrator rollback``` - Rolls back the last applied migration.

```migrator rollback all``` - Rolls back all migration.

```migrator help``` - Print commands.