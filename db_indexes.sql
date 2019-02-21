CREATE UNIQUE INDEX idx_block_hash ON block (hash);
CREATE UNIQUE INDEX idx_block_nextblock_id ON block (nextblock_id);
CREATE INDEX idx_address_balance ON address (balance);
CREATE UNIQUE INDEX idx_transaction_hash ON tx (hash);
CREATE INDEX idx_transaction_locktime ON tx (locktime);
CREATE INDEX idx_transaction_fee ON tx (fee);
CREATE INDEX idx_transaction_size ON tx (size);
CREATE INDEX idx_transaction_vsize ON tx (vsize);
CREATE INDEX idx_transaction_block_hash ON tx (block_hash);
# add indexes for tx_input and tx_output here
CREATE INDEX idx_props_property ON props (property);

