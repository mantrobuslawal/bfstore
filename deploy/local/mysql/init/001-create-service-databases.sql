-- bfstore local MySQL bootstrap
-- Creates service-owned databases for local development.
-- Runs only when the MySQL data directory is initialised from a fresh volume.

CREATE DATABASE IF NOT EXISTS bfstore_catalog
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

CREATE DATABASE IF NOT EXISTS bfstore_basket
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

CREATE DATABASE IF NOT EXISTS bfstore_inventory
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

CREATE DATABASE IF NOT EXISTS bfstore_order
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

CREATE DATABASE IF NOT EXISTS bfstore_payment
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
