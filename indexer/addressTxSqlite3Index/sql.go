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