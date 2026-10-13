-- Run once in the SQL Editor.
-- A separate schema keeps these tables outside the usual public Data API schema.
-- create a schema named backend_labs, unless it already exists.
-- A schema is a namespace that groups tables and other database objects, somewhat like a folder.
CREATE SCHEMA IF NOT EXISTS backend_labs;

-- our organization looks like:
-- Database: neondb
-- └── Schema: backend_labs
--     └── Table: notes


-- Inside the backend_labs schema, create a table called notes, unless it already exists.
-- parentheses contains table's column definition
CREATE TABLE IF NOT EXISTS backend_labs.notes (
    -- PostgreSQL supplies the ID when we insert a note without providing one.
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    -- Create a column named text, with PostgreSQL’s text data type. It cannot be null, and it must contain something besides ordinary spaces.
    -- btrim -> remove ordinary spaces from both ends of the value for this check
    text text NOT NULL CHECK (length(btrim(text)) > 0)
);
