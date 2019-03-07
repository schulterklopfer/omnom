/*
 * MIT License
 *
 * Copyright (c) 2019 schulterklopfer/SKP
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILIT * Y, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package fullSqlite3Index

const SQLInsertBlock = "INSERT INTO block(hash,prevblock_id,nextblock_id,version,blocktime) VALUES(?,?,?,?,?);"
const SQLInsertTx = "INSERT INTO tx(txid,block_id,hash,locktime,size,vsize,weight,base_size) VALUES(?,?,?,?,?,?,?,?);"
const SQLUpdateTxFeeAmount = "UPDATE tx SET fee=?, amount=? WHERE id=?;"
const SQLUpsertAddress = "INSERT INTO address(address,balance) VALUES(?,?) ON CONFLICT(address) DO UPDATE SET balance=balance+excluded.balance;"

const SQLUpdateAddressBalance = "UPDATE address SET balance=balance+? WHERE id=?;"
const SQLInsertInput = "INSERT INTO tx_input(tx_id,output_id) VALUES(?,?);"
const SQLInsertOutput = "INSERT INTO tx_output(tx_id,idx,amount,address_id) VALUES(?,?,?,?);"

const SQLSelectOutput = "SELECT o.id, o.amount, o.address_id FROM tx LEFT JOIN tx_output o ON tx.id = o.tx_id LEFT JOIN address a on o.address_id = a.id WHERE txid=? AND o.idx=?"

const SQLOnStart = `PRAGMA foreign_keys = OFF;

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
`

const SQLOnEnd = `
CREATE UNIQUE INDEX idx_block_hash ON block (hash);
CREATE UNIQUE INDEX idx_block_nextblock_id ON block (nextblock_id);
CREATE INDEX idx_address_balance ON address (balance);
CREATE UNIQUE INDEX idx_transaction_hash ON tx (hash);
CREATE INDEX idx_transaction_locktime ON tx (locktime);
CREATE INDEX idx_transaction_fee ON tx (fee);
CREATE INDEX idx_transaction_size ON tx (size);
CREATE INDEX idx_transaction_vsize ON tx (vsize);
CREATE INDEX idx_transaction_block_hash ON tx (block_hash);
CREATE INDEX idx_props_property ON props (property);
`
