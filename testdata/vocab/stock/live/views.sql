-- The relations the connectors beside this file read (ADR 0020), as views
-- over a synthetic parts system's and yard system's own tables. nitrovin
-- dictates the views' column names, never their SQL: each system keeps its
-- tables, and a view renames. pkg/stock/pg's TestParityPostgres runs this
-- file, fills the tables with the rows of testdata/vocab/stock's files, and
-- answers fit from both. docs/vocabularies.md (Connectors) walks through it.

-- A parts system: two shelves in one table, and the catalog release each
-- is numbered under.
CREATE SCHEMA parts;
CREATE TABLE parts.shelf (
    shelf       text NOT NULL,             -- 'main' or 'annex'
    stock_id    text PRIMARY KEY,
    bin         text NOT NULL,
    part_class  text,                      -- the fitment key, for a numbered part
    catalog_no  text,
    hand        text CHECK (hand IN ('left', 'right', 'both')),
    spine_part  text,                      -- the part type, for a part with no number
    make        text,                      -- and its vehicle and year
    model       text,
    model_year  integer,
    attrs       jsonb NOT NULL DEFAULT '{}'
);
CREATE INDEX shelf_by_number ON parts.shelf (part_class, catalog_no, hand);
CREATE INDEX shelf_by_vehicle ON parts.shelf (spine_part, make, model, model_year) WHERE catalog_no IS NULL;
CREATE TABLE parts.release (
    shelf   text PRIMARY KEY,
    release text NOT NULL
);

-- A yard system: whole vehicles, each with the vPIC decode it computed
-- for its VIN (by nitrovin's batch decode) and its own make and model.
CREATE SCHEMA yard;
CREATE TABLE yard.vehicles (
    stock_id   text PRIMARY KEY,
    lot        text NOT NULL,
    vpic_make  integer,
    vpic_model integer,
    vpic_year  integer,
    make       text,
    model      text,
    model_year integer,
    attrs      jsonb NOT NULL DEFAULT '{}'
);
CREATE INDEX yard_by_model ON yard.vehicles (vpic_model, vpic_year);
CREATE INDEX yard_by_vehicle ON yard.vehicles (make, model, model_year);

-- The views. A shelf has type, number and side (side null for a fitment
-- with none; on a row with no number, type and side are not read),
-- part_type, vehicle_<field> per field of the catalog's vehicle
-- vocabulary (catalog-models: make, model), year and fields; a yard has
-- decoded_make, decoded_model and decoded_year in place of the shelf's
-- part. Either may add as_of, a row's own time.
--
-- Each is a security_barrier view: its WHERE runs before any condition of
-- the query reading it that gives a column to a function not marked
-- leakproof, so a function the reader writes, which may raise or log its
-- arguments, never sees a row the view leaves out. A plain view lets the
-- planner run a cheap condition first. The connectors' conditions give
-- these columns to leakproof functions only, so the barrier passes them
-- to the indexes above, which TestPlansPostgres holds them to.
CREATE SCHEMA stock;
CREATE VIEW stock.shelf_stock WITH (security_barrier) AS
SELECT stock_id AS id, bin AS location,
       part_class AS type, catalog_no AS number, hand AS side, spine_part AS part_type,
       make AS vehicle_make, model AS vehicle_model, model_year AS year,
       attrs AS fields
FROM parts.shelf WHERE shelf = 'main';
CREATE VIEW stock.shelf_stock_release WITH (security_barrier) AS
SELECT release FROM parts.release WHERE shelf = 'main';
CREATE VIEW stock.shelf_annex WITH (security_barrier) AS
SELECT stock_id AS id, bin AS location,
       part_class AS type, catalog_no AS number, hand AS side, spine_part AS part_type,
       make AS vehicle_make, model AS vehicle_model, model_year AS year,
       attrs AS fields
FROM parts.shelf WHERE shelf = 'annex';
CREATE VIEW stock.yard_stock WITH (security_barrier) AS
SELECT stock_id AS id, lot AS location,
       vpic_make AS decoded_make, vpic_model AS decoded_model, vpic_year AS decoded_year,
       make AS vehicle_make, model AS vehicle_model, model_year AS year,
       attrs AS fields
FROM yard.vehicles;

-- The connection's user reads the views and nothing else.
--
-- It can create no function: a database made by Postgres 15 or later
-- grants PUBLIC no CREATE on schema public (revoke it on an older one),
-- and TEMPORARY, which every role holds through PUBLIC and would let it
-- create one in pg_temp, is revoked from PUBLIC on the database.
-- Making the role NOINHERIT would not do: PUBLIC's privileges reach every
-- role whatever it inherits. Grant TEMPORARY back to a role that needs
-- temporary tables.
--
-- GRANT ... ON ALL TABLES covers the views that exist when it runs; the
-- default privileges grant SELECT on those this script's role makes in
-- stock later.
CREATE ROLE nitrovin_stock LOGIN PASSWORD 'stock-reader';
DO $$ BEGIN
  EXECUTE format('REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC', current_database());
END $$;
GRANT USAGE ON SCHEMA stock TO nitrovin_stock;
GRANT SELECT ON ALL TABLES IN SCHEMA stock TO nitrovin_stock;
ALTER DEFAULT PRIVILEGES IN SCHEMA stock GRANT SELECT ON TABLES TO nitrovin_stock;
