PRAGMA foreign_keys = ON;

CREATE TABLE address (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    address TEXT,
    balance INTEGER,
    inserted_ts INTEGER DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_address_address ON address (address);

CREATE TABLE tx (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    txid TEXT,
    block_hash TEXT,
    hash TEXT,
    timereceived INTEGER,
    amount INTEGER,
    fee INTEGER,
    size INTEGER,
    vsize INTEGER,
    inserted_ts INTEGER DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE address_tx (
    address_id INTEGER REFERENCES address,
    transaction_id INTEGER REFERENCES tx
);

CREATE TABLE props (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  property TEXT,
  value TEXT,
  inserted_ts INTEGER DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO props (property, value) VALUES ("blockheight", "-1");
