# GATOR
---

**These are very rough notes while building the app.**

- you would need to start the PostgreSQL service inside a new container, *everytime*.
	- `service postgresql start`
	- run `passwd postgres` then type in the system-wide password.
- you would also need to setup a DB.
	- `sudo -u postgres psql`
	- `CREATE DATABASE gator;`
	- `\c gator` -> to connect to the DB.
	- once, this DB is created, you can use:
	- `sudo -u postgres psql gator` to directly go into the DB.
	- `ALTER USER postgres PASSWORD 'postgres';` -> to change user level passwd, this will be used in the connection string later.
- creating tables is taken care of by the code.
- `\dt` will show a list of tables
- `\q` -> quit from the PostgreSQL CLI.
- apologies for the same, bear w/ it until I find a robust solution to it.
---

- the connection string you pass to "Goose" needs to be of the form:
	- `goose postgres "postgres://username:passwd@localhost:5432/gator" up/down`
- the command to generate Go code from SQL-ish files is `sqlc generate`. It needs to have a YAML file describing what needs to be done. I already have made it part of the repo. Feel free to make changes to adapt to your environment.
- both packages need to be installed into your machine first: *sqlc* and *goose*.

