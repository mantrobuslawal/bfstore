-- bfstore local MySQL bootstrap
--
-- This file creates local service users and grants each service user access
-- only to its own service-owned database.
--
-- The host is '%' for local Docker development because connections may come
-- from the Docker bridge gateway, not from localhost.
--
-- The CREATE USER + ALTER USER pattern is intentional:
--
--   CREATE USER IF NOT EXISTS
--     ensures the user exists
--
--   ALTER USER
--     ensures the local password is reset to the expected deterministic value
--
-- This keeps local database bootstrap idempotent and repeatable after partial
-- setup attempts or stale local volumes.

CREATE USER IF NOT EXISTS 'bfstore_catalog'@'%'
  IDENTIFIED BY 'bfstore_catalog_password';

ALTER USER 'bfstore_catalog'@'%'
  IDENTIFIED BY 'bfstore_catalog_password';

CREATE USER IF NOT EXISTS 'bfstore_basket'@'%'
  IDENTIFIED BY 'bfstore_basket_password';

ALTER USER 'bfstore_basket'@'%'
  IDENTIFIED BY 'bfstore_basket_password';

CREATE USER IF NOT EXISTS 'bfstore_inventory'@'%'
  IDENTIFIED BY 'bfstore_inventory_password';

ALTER USER 'bfstore_inventory'@'%'
  IDENTIFIED BY 'bfstore_inventory_password';

CREATE USER IF NOT EXISTS 'bfstore_order'@'%'
  IDENTIFIED BY 'bfstore_order_password';

ALTER USER 'bfstore_order'@'%'
  IDENTIFIED BY 'bfstore_order_password';

CREATE USER IF NOT EXISTS 'bfstore_payment'@'%'
  IDENTIFIED BY 'bfstore_payment_password';

ALTER USER 'bfstore_payment'@'%'
  IDENTIFIED BY 'bfstore_payment_password';

GRANT ALL PRIVILEGES ON bfstore_catalog.*
  TO 'bfstore_catalog'@'%';

GRANT ALL PRIVILEGES ON bfstore_basket.*
  TO 'bfstore_basket'@'%';

GRANT ALL PRIVILEGES ON bfstore_inventory.*
  TO 'bfstore_inventory'@'%';

GRANT ALL PRIVILEGES ON bfstore_order.*
  TO 'bfstore_order'@'%';

GRANT ALL PRIVILEGES ON bfstore_payment.*
  TO 'bfstore_payment'@'%';

FLUSH PRIVILEGES;
