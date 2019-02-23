package addressTxSqlite3Index

const SQLInsertTx = "INSERT INTO tx(txid,hash,blockhash,locktime,size,vsize,weight,base_size) VALUES(?,?,?,?,?,?,?,?);"
const SQLInsertAddress = "INSERT INTO address(address) VALUES(?);"
const SQLSelectAddress = "SELECT id FROM address WHERE address=?;"
const SQLInsertTxAddress = "INSERT INTO tx_address(tx_id,address_id) VALUES(?,?);"

const SQLOnStart = `PRAGMA foreign_keys = OFF;

CREATE TABLE tx (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    txid TEXT,
    hash TEXT,
    blockhash TEXT,
    locktime INTEGER,
    amount INTEGER,
    fee INTEGER,
    size INTEGER,
    vsize INTEGER,
    weight INTEGER,
    base_size INTEGER
);

CREATE TABLE address (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
	address TEXT
);

CREATE TABLE tx_address (
    tx_id INTEGER REFERENCES tx,
	address_id INTEGER REFERENCES address
);

CREATE UNIQUE INDEX address_address ON address (address);
`


const SQLOnEnd = `
CREATE INDEX idx_address_tx_id ON tx_address (tx_id);
CREATE INDEX idx_address_address_id ON tx_address (address_id);

CREATE UNIQUE INDEX idx_transaction_txid ON tx (txid);
CREATE UNIQUE INDEX idx_transaction_hash ON tx (hash);
CREATE INDEX idx_transaction_locktime ON tx (locktime);
CREATE INDEX idx_transaction_size ON tx (size);
CREATE INDEX idx_transaction_vsize ON tx (vsize);
CREATE INDEX idx_transaction_blockhash ON tx (blockhash);
`