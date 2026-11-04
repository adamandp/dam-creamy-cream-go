INSERT INTO roles (id,name,created_at,updated_at)
VALUES ('11111111-1111-4111-8111-111111111111','Customer','2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
INSERT INTO users (id,role_id,username,password,full_name,is_active,created_at,updated_at)
VALUES ('22222222-2222-4222-8222-222222222222','11111111-1111-4111-8111-111111111111','PromoFixture','fixture-not-a-login','Promo Fixture',true,'2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
INSERT INTO category (id,name,is_active,created_at,updated_at)
VALUES ('33333333-3333-4333-8333-333333333333','Ice Cream',true,'2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
INSERT INTO products (id,category_id,name,price,is_active,created_at,updated_at)
VALUES ('44444444-4444-4444-8444-444444444444','33333333-3333-4333-8333-333333333333','Strawberry',25000,true,'2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
INSERT INTO discounts (id,name,discount_type,value,max_applied,max_redeemed,redeemed_count,description,product_id,quantity,start_date,end_date,created_at,updated_at)
VALUES ('55555555-5555-4555-8555-555555555555','Winter Deal','PERCENTAGE',20,2,100,0,NULL,NULL,NULL,'2026-01-01 00:00:00+00','2027-01-01 00:00:00+00','2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
INSERT INTO product_discounts (id,product_id,discount_id,is_active,created_at,updated_at)
VALUES ('66666666-6666-4666-8666-666666666666','44444444-4444-4444-8444-444444444444','55555555-5555-4555-8555-555555555555',true,'2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
INSERT INTO vouchers (id,code,name,description,type,"group",value,product_id,quantity,minimum_purchase,is_active,start_date,end_date,max_redeemed,redeemed_count,created_at,updated_at)
VALUES ('77777777-7777-4777-8777-777777777777','WELCOME20','Welcome Deal',NULL,'PERCENTAGE','CHECKOUT',20,NULL,NULL,50000,true,'2026-01-01 00:00:00+00','2027-01-01 00:00:00+00',100,0,'2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
INSERT INTO user_vouchers (id,user_id,voucher_id,status,claimed_at,expired_at,created_at,updated_at)
VALUES ('88888888-8888-4888-8888-888888888888','22222222-2222-4222-8222-222222222222','77777777-7777-4777-8777-777777777777','CLAIMED','2026-01-01 00:00:00+00','2027-01-01 00:00:00+00','2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');

-- Deliberately add unrelated users. The NestJS Product Discounts list counts
-- users, whereas the corrected Go endpoint counts product discount records.
INSERT INTO roles(id,name,created_at,updated_at)
VALUES ('99999999-9999-4999-8999-999999999999','Admin','2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
INSERT INTO users(id,role_id,username,password,full_name,is_active,created_at,updated_at)
VALUES
('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','99999999-9999-4999-8999-999999999999','ParityAdmin','fixture-not-a-login','Parity Admin',true,'2026-01-01 00:00:00+00','2026-01-01 00:00:00+00'),
('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','11111111-1111-4111-8111-111111111111','ParityOther','fixture-not-a-login','Parity Other',true,'2026-01-01 00:00:00+00','2026-01-01 00:00:00+00');
