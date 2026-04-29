ALTER TABLE orders DROP CONSTRAINT unique_user_id_order_number;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS users;