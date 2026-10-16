CREATE SCHEMA IF NOT EXISTS url_shortener;
CREATE TABLE IF NOT EXISTS url_shortener.links (
    code varchar(12) PRIMARY KEY,
    original_url text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
