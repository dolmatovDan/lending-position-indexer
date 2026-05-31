CREATE TABLE IF NOT EXISTS positions (
    protocol        TEXT        NOT NULL,
    wallet_address  TEXT        NOT NULL,
    market_id       TEXT        NOT NULL,
    token_address   TEXT        NOT NULL,
    token_symbol    TEXT        NOT NULL,
    token_decimals  SMALLINT    NOT NULL,
    side            TEXT        NOT NULL,
    amount          NUMERIC     NOT NULL,
    price           NUMERIC     NOT NULL,
    health_factor   NUMERIC     NOT NULL,
    block_number    BIGINT      NOT NULL,
    block_timestamp TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (protocol, wallet_address, market_id, token_address, side, block_number)
);

CREATE INDEX IF NOT EXISTS idx_positions_wallet_block
    ON positions (wallet_address, block_number DESC);

CREATE INDEX IF NOT EXISTS idx_positions_wallet_protocol_block
    ON positions (wallet_address, protocol, block_number DESC);
