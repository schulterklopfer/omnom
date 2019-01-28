#!/bin/bash

DB_FILE="index.sqlite"

if [ -e ${DB_FILE} ]; then
	echo "DB found, creating indexes"
	cat db_indexes.sql | sqlite3 $DB_FILE
fi