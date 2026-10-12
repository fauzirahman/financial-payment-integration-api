-- Demo data for the UUID financial schema and migration 008.
-- Prerequisites: migrations/008_payment_journals_and_reporting.sql and
-- migrations/003_create_webhook_events.sql (for public.webhook_events).
-- Safe to re-run: stable natural keys and ON CONFLICT DO NOTHING are used.

BEGIN;

-- A dedicated demo customer, so this seed does not depend on production data.
INSERT INTO public.customers (
    id, customer_number, name, email, phone, status, created_at, updated_at
)
VALUES (
    '00000000-0000-0000-0000-000000000101'::uuid,
    'CUST-DEMO-001', 'Demo Customer', 'demo.customer@example.test',
    '+620000000001', 'ACTIVE', NOW(), NOW()
)
ON CONFLICT (customer_number) DO NOTHING;

INSERT INTO public.products (sku, name, description, unit, is_stocked, is_active)
VALUES
    ('SKU-DEMO-001', 'Demo Widget', 'Seed item for sales and inventory flow', 'UNIT', TRUE, TRUE),
    ('SKU-DEMO-002', 'Demo Cable', 'Second seed item for delivery flow', 'UNIT', TRUE, TRUE)
ON CONFLICT (sku) DO NOTHING;

INSERT INTO public.warehouses (code, name, address, is_active)
VALUES
    ('WH-DEMO-JKT', 'Demo Jakarta Warehouse', '{"city":"Jakarta","country":"ID"}'::jsonb, TRUE)
ON CONFLICT (code) DO NOTHING;

INSERT INTO public.inventory_balances (product_id, warehouse_id, on_hand, reserved)
SELECT p.id, w.id, seed.on_hand, seed.reserved
FROM (VALUES
    ('SKU-DEMO-001', 100::numeric, 2::numeric),
    ('SKU-DEMO-002', 50::numeric, 1::numeric)
) AS seed(sku, on_hand, reserved)
JOIN public.products p ON p.sku = seed.sku
JOIN public.warehouses w ON w.code = 'WH-DEMO-JKT'
ON CONFLICT (product_id, warehouse_id) DO NOTHING;

INSERT INTO public.sales_rfqs (
    rfq_number, customer_id, source_channel, source_reference, subject, description, status
)
SELECT 'RFQ-DEMO-000001', id, 'WEBSITE', 'demo-web-001', 'Demo widget order',
       'Seed RFQ for the demo sales flow', 'QUOTED'
FROM public.customers WHERE customer_number = 'CUST-DEMO-001'
ON CONFLICT (rfq_number) DO NOTHING;

INSERT INTO public.sales_quotations (
    quotation_number, rfq_id, customer_id, revision, status, currency,
    valid_until, subtotal, tax_total, total, terms, notes
)
SELECT 'QUO-DEMO-000001', r.id, c.id, 1, 'ACCEPTED', 'IDR', CURRENT_DATE + 30,
       250000, 0, 250000, 'Demo terms', 'Seed quotation'
FROM public.sales_rfqs r
JOIN public.customers c ON c.customer_number = 'CUST-DEMO-001'
WHERE r.rfq_number = 'RFQ-DEMO-000001'
ON CONFLICT (quotation_number) DO NOTHING;

INSERT INTO public.sales_quotation_items (
    quotation_id, line_number, description, sku, quantity, unit, unit_price, tax_rate, line_total
)
SELECT q.id, 1, 'Demo Widget', 'SKU-DEMO-001', 2, 'UNIT', 100000, 0, 200000
FROM public.sales_quotations q WHERE q.quotation_number = 'QUO-DEMO-000001'
ON CONFLICT (quotation_id, line_number) DO NOTHING;

INSERT INTO public.sales_quotation_items (
    quotation_id, line_number, description, sku, quantity, unit, unit_price, tax_rate, line_total
)
SELECT q.id, 2, 'Demo Cable', 'SKU-DEMO-002', 1, 'UNIT', 50000, 0, 50000
FROM public.sales_quotations q WHERE q.quotation_number = 'QUO-DEMO-000001'
ON CONFLICT (quotation_id, line_number) DO NOTHING;

INSERT INTO public.sales_orders (
    order_number, quotation_id, customer_id, status, currency, subtotal, tax_total,
    total, payment_terms, delivery_address, confirmed_at
)
SELECT 'SO-DEMO-000001', q.id, c.id, 'CONFIRMED', 'IDR', 250000, 0,
       250000, 'Demo payment terms', '{"city":"Jakarta","country":"ID"}'::jsonb, NOW()
FROM public.sales_quotations q
JOIN public.customers c ON c.customer_number = 'CUST-DEMO-001'
WHERE q.quotation_number = 'QUO-DEMO-000001'
ON CONFLICT (order_number) DO NOTHING;

INSERT INTO public.sales_order_items (
    sales_order_id, line_number, product_id, sku, description, quantity, unit,
    unit_price, tax_rate, line_total
)
SELECT so.id, 1, p.id, p.sku, p.name, 2, 'UNIT', 100000, 0, 200000
FROM public.sales_orders so
JOIN public.products p ON p.sku = 'SKU-DEMO-001'
WHERE so.order_number = 'SO-DEMO-000001'
ON CONFLICT (sales_order_id, line_number) DO NOTHING;

INSERT INTO public.sales_order_items (
    sales_order_id, line_number, product_id, sku, description, quantity, unit,
    unit_price, tax_rate, line_total
)
SELECT so.id, 2, p.id, p.sku, p.name, 1, 'UNIT', 50000, 0, 50000
FROM public.sales_orders so
JOIN public.products p ON p.sku = 'SKU-DEMO-002'
WHERE so.order_number = 'SO-DEMO-000001'
ON CONFLICT (sales_order_id, line_number) DO NOTHING;

INSERT INTO public.sales_invoices (
    invoice_number, sales_order_id, customer_id, status, currency, subtotal,
    tax_total, total, amount_paid, issued_at, due_date
)
SELECT 'INV-DEMO-000001', so.id, c.id, 'ISSUED', 'IDR', 250000,
       0, 250000, 0, NOW(), CURRENT_DATE + 14
FROM public.sales_orders so
JOIN public.customers c ON c.customer_number = 'CUST-DEMO-001'
WHERE so.order_number = 'SO-DEMO-000001'
ON CONFLICT (invoice_number) DO NOTHING;

INSERT INTO public.sales_invoice_items (
    invoice_id, sales_order_item_id, line_number, description, quantity,
    unit_price, tax_rate, line_total
)
SELECT i.id, soi.id, soi.line_number, soi.description, soi.quantity,
       soi.unit_price, soi.tax_rate, soi.line_total
FROM public.sales_invoices i
JOIN public.sales_orders so ON so.order_number = 'SO-DEMO-000001'
JOIN public.sales_order_items soi ON soi.sales_order_id = so.id
WHERE i.invoice_number = 'INV-DEMO-000001'
ON CONFLICT (invoice_id, line_number) DO NOTHING;

-- Reserve the ordered quantity for each product in the demo order.
INSERT INTO public.inventory_reservations (
    sales_order_item_id, product_id, warehouse_id, quantity, status
)
SELECT soi.id, soi.product_id, w.id, soi.quantity, 'RESERVED'
FROM public.sales_orders so
JOIN public.sales_order_items soi ON soi.sales_order_id = so.id
JOIN public.warehouses w ON w.code = 'WH-DEMO-JKT'
WHERE so.order_number = 'SO-DEMO-000001'
  AND NOT EXISTS (
      SELECT 1 FROM public.inventory_reservations ir
      WHERE ir.sales_order_item_id = soi.id
        AND ir.product_id = soi.product_id
        AND ir.warehouse_id = w.id
        AND ir.status = 'RESERVED'
  );

INSERT INTO public.delivery_orders (
    delivery_number, sales_order_id, warehouse_id, status, recipient_name,
    delivery_address, notes, processed_at
)
SELECT 'DO-DEMO-000001', so.id, w.id, 'DRAFT', 'Demo Customer',
       '{"city":"Jakarta","country":"ID"}'::jsonb,
       'Seed delivery order; no inventory movement is recorded until dispatch.', NULL
FROM public.sales_orders so
JOIN public.warehouses w ON w.code = 'WH-DEMO-JKT'
WHERE so.order_number = 'SO-DEMO-000001'
ON CONFLICT (delivery_number) DO NOTHING;

INSERT INTO public.delivery_order_items (delivery_order_id, sales_order_item_id, quantity)
SELECT d.id, soi.id, soi.quantity
FROM public.delivery_orders d
JOIN public.sales_orders so ON so.order_number = 'SO-DEMO-000001'
JOIN public.sales_order_items soi ON soi.sales_order_id = so.id
WHERE d.delivery_number = 'DO-DEMO-000001'
ON CONFLICT (delivery_order_id, sales_order_item_id) DO NOTHING;

-- Immutable movement history: seed opening balances once, without changing stock.
INSERT INTO public.inventory_movements (
    product_id, warehouse_id, movement_type, quantity_delta,
    reference_type, reference_id, occurred_at
)
SELECT p.id, w.id, 'OPENING', seed.quantity,
       'SEED', '00000000-0000-0000-0000-000000000801'::uuid, TIMESTAMP '2026-01-01 00:00:00'
FROM (VALUES ('SKU-DEMO-001', 100::numeric), ('SKU-DEMO-002', 50::numeric)) AS seed(sku, quantity)
JOIN public.products p ON p.sku = seed.sku
JOIN public.warehouses w ON w.code = 'WH-DEMO-JKT'
ON CONFLICT (movement_type, reference_type, reference_id, product_id, warehouse_id) DO NOTHING;

-- This is the legacy webhook_events table from migration 003.
INSERT INTO public.webhook_events (event_id, payment_reference, event_type)
VALUES
    ('evt-demo-payment-success-001', 'PAY-DEMO-000001', 'payment.success'),
    ('evt-demo-payment-failed-001', 'PAY-DEMO-000002', 'payment.failed')
ON CONFLICT (event_id) DO NOTHING;

COMMIT;
