#!/bin/bash

DB_FILE="index.sqlite"

if [ ! -e ${DB_FILE} ]; then
	echo "DB not found, creating..."
	cat db.sql | sqlite3 $DB_FILE
fi