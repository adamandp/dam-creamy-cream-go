-- Additional deterministic rows for the final five NestJS domains.
-- Apply to both disposable databases after promotions.sql, never production.
INSERT INTO category (id,name,is_active,created_at,updated_at)
VALUES ('cccccccc-cccc-4ccc-8ccc-cccccccccccc','Classic Scoops',true,'2026-01-01','2026-01-01');

INSERT INTO products (id,category_id,name,price,is_active,created_at,updated_at)
VALUES ('dddddddd-dddd-4ddd-8ddd-dddddddddddd','cccccccc-cccc-4ccc-8ccc-cccccccccccc','Classic Vanilla',12000,true,'2026-01-01','2026-01-01');

INSERT INTO reviews (id,user_id,product_id,rating,comment,created_at,updated_at)
VALUES
('e1111111-1111-4111-8111-111111111111','22222222-2222-4222-8222-222222222222','44444444-4444-4444-8444-444444444444',5,'Great','2026-01-01','2026-01-01'),
('e2222222-2222-4222-8222-222222222222','bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','dddddddd-dddd-4ddd-8ddd-dddddddddddd',4,'Classic','2026-01-01','2026-01-01');

INSERT INTO product_analytics (id,user_id,product_id,created_at)
VALUES
('e3333333-3333-4333-8333-333333333333','22222222-2222-4222-8222-222222222222','44444444-4444-4444-8444-444444444444','2026-01-01'),
('e4444444-4444-4444-8444-444444444444','bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','dddddddd-dddd-4ddd-8ddd-dddddddddddd','2026-01-01');

INSERT INTO notifications (id,title,message,created_at,updated_at)
VALUES
('e5555555-5555-4555-8555-555555555555','Delivery','Order is ready','2026-01-01','2026-01-01'),
('e6666666-6666-4666-8666-666666666666','Marketing','Try our new flavor','2026-01-01','2026-01-01');

INSERT INTO user_notifications (id,user_id,notification_id,is_read,created_at,updated_at)
VALUES ('e7777777-7777-4777-8777-777777777777','22222222-2222-4222-8222-222222222222','e5555555-5555-4555-8555-555555555555',false,'2026-01-01','2026-01-01');
