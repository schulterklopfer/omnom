PRAGMA foreign_keys = ON;

CREATE TABLE block (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hash TEXT,
    prevblock_id INTEGER REFERENCES block,
    nextblock_id INTEGER REFERENCES block,
    version INTEGER,
    blocktime INTEGER
);

CREATE TABLE tx (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    block_id INTEGER REFERENCES block,
    txid TEXT,
    hash TEXT,
    locktime INTEGER,
    amount INTEGER,
    fee INTEGER,
    size INTEGER,
    vsize INTEGER,
    weight INTEGER,
    base_size INTEGER
);

CREATE UNIQUE INDEX idx_transaction_txid ON tx (txid);

CREATE TABLE tx_input (
    tx_id INTEGER REFERENCES tx,
    output_id INTEGER REFERENCES tx_output
);

CREATE TABLE tx_output (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tx_id INTEGER REFERENCES tx,
    idx INTEGER,
    amount INTEGER,
    address_id INTEGER REFERENCES address
);

CREATE INDEX idx_tx_output_tx_id ON tx_output (tx_id);
CREATE INDEX idx_tx_output_idx ON tx_output (idx);
CREATE INDEX idx_tx_address_id ON tx_output (address_id);


CREATE TABLE address (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    address TEXT,
    balance INTEGER
);

CREATE UNIQUE INDEX idx_address_address ON address (address);

CREATE TABLE props (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  property TEXT,
  value TEXT
);

INSERT INTO props (property, value) VALUES ("blockheight", "-1");



