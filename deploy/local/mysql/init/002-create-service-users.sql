-- bfstore local MySQL bootstrap
-- Creates local service users and grants each service user access only to its own database.
-- Host '%' is used for local Docker development because connections may come from the Docker bridge gateway.

CREATE USER IF NOT EXISTS 'bfstore_catalog'@'%'
  IDENTIFIED BY 'bfstore_catalog_password';

CREATE USER IF NOT EXISTS 'bfstore_basket'@'%'
  IDENTIFIED BY 'bfstore_basket_password';

CREATE USER IF NOT EXISTS 'bfstore_inventory'@'%'
  IDENTIFIED BY 'bfstore_inventory_password';

CREATE USER IF NOT EXISTS 'bfstore_order'@'%'
  IDENTIFIED BY 'bfstore_order_password';

CREATE USER IF NOT EXISTS 'bfstore_payment'@'%'
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
