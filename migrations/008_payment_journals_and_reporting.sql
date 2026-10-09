BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS public.document_sequences (
    document_type VARCHAR(40) PRIMARY KEY,
    prefix VARCHAR(20) NOT NULL,
    next_value BIGINT NOT NULL DEFAULT 1 CHECK (next_value > 0),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION public.next_document_number(sequence_type VARCHAR)
RETURNS VARCHAR
LANGUAGE plpgsql
AS $$
DECLARE
    sequence_prefix VARCHAR(20);
    sequence_value BIGINT;
BEGIN
    INSERT INTO public.document_sequences (document_type, prefix, next_value)
    VALUES (sequence_type, upper(sequence_type), 2)
    ON CONFLICT (document_type) DO UPDATE
        SET next_value = public.document_sequences.next_value + 1,
            updated_at = NOW()
    RETURNING prefix, next_value - 1 INTO sequence_prefix, sequence_value;

    RETURN sequence_prefix || '-' || TO_CHAR(CURRENT_DATE, 'YYYY') || '-' || LPAD(sequence_value::TEXT, 6, '0');
END;
$$;

CREATE TABLE IF NOT EXISTS public.sales_rfqs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rfq_number VARCHAR(50) NOT NULL UNIQUE DEFAULT public.next_document_number('rfq'),
    customer_id UUID NOT NULL REFERENCES public.customers(id) ON DELETE RESTRICT,
    source_channel VARCHAR(20) NOT NULL CHECK (source_channel IN ('WEBSITE', 'WHATSAPP', 'EMAIL', 'PHONE', 'OTHER')),
    source_reference VARCHAR(255),
    subject VARCHAR(200) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'QUOTED', 'WON', 'LOST', 'CANCELLED')),
    received_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.sales_quotations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_number VARCHAR(50) NOT NULL UNIQUE DEFAULT public.next_document_number('quotation'),
    rfq_id UUID REFERENCES public.sales_rfqs(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES public.customers(id) ON DELETE RESTRICT,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ISSUED', 'ACCEPTED', 'REJECTED', 'EXPIRED', 'SUPERSEDED', 'CANCELLED')),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    valid_until DATE,
    subtotal NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (subtotal >= 0),
    tax_total NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (tax_total >= 0),
    total NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (total >= 0),
    terms TEXT,
    notes TEXT,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (rfq_id, revision)
);

CREATE TABLE IF NOT EXISTS public.sales_quotation_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_id UUID NOT NULL REFERENCES public.sales_quotations(id) ON DELETE RESTRICT,
    line_number INTEGER NOT NULL CHECK (line_number > 0),
    description TEXT NOT NULL,
    sku VARCHAR(60),
    quantity NUMERIC(20,4) NOT NULL CHECK (quantity > 0),
    unit VARCHAR(20) NOT NULL DEFAULT 'UNIT',
    unit_price NUMERIC(20,2) NOT NULL CHECK (unit_price >= 0),
    tax_rate NUMERIC(7,4) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
    line_total NUMERIC(20,2) NOT NULL CHECK (line_total >= 0),
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (quotation_id, line_number)
);

CREATE TABLE IF NOT EXISTS public.sales_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(50) NOT NULL UNIQUE DEFAULT public.next_document_number('sales-order'),
    quotation_id UUID REFERENCES public.sales_quotations(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES public.customers(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'CONFIRMED', 'PARTIALLY_DELIVERED', 'DELIVERED', 'CANCELLED')),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    subtotal NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (subtotal >= 0),
    tax_total NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (tax_total >= 0),
    total NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (total >= 0),
    payment_terms TEXT,
    delivery_address JSONB NOT NULL DEFAULT '{}'::JSONB,
    confirmed_at TIMESTAMP(0) WITHOUT TIME ZONE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.sales_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_order_id UUID NOT NULL REFERENCES public.sales_orders(id) ON DELETE RESTRICT,
    line_number INTEGER NOT NULL CHECK (line_number > 0),
    product_id UUID,
    sku VARCHAR(60),
    description TEXT NOT NULL,
    quantity NUMERIC(20,4) NOT NULL CHECK (quantity > 0),
    unit VARCHAR(20) NOT NULL DEFAULT 'UNIT',
    unit_price NUMERIC(20,2) NOT NULL CHECK (unit_price >= 0),
    tax_rate NUMERIC(7,4) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
    line_total NUMERIC(20,2) NOT NULL CHECK (line_total >= 0),
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (sales_order_id, line_number)
);

CREATE TABLE IF NOT EXISTS public.proforma_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proforma_number VARCHAR(50) NOT NULL UNIQUE DEFAULT public.next_document_number('proforma'),
    quotation_id UUID REFERENCES public.sales_quotations(id) ON DELETE RESTRICT,
    sales_order_id UUID REFERENCES public.sales_orders(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES public.customers(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ISSUED', 'SUPERSEDED', 'CANCELLED')),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    subtotal NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (subtotal >= 0),
    tax_total NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (tax_total >= 0),
    total NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (total >= 0),
    requested_deposit NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (requested_deposit >= 0 AND requested_deposit <= total),
    issued_at TIMESTAMP(0) WITHOUT TIME ZONE,
    expires_at DATE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (quotation_id IS NOT NULL OR sales_order_id IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS public.sales_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(50) NOT NULL UNIQUE DEFAULT public.next_document_number('invoice'),
    sales_order_id UUID NOT NULL REFERENCES public.sales_orders(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES public.customers(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ISSUED', 'PARTIALLY_PAID', 'PAID', 'OVERDUE', 'VOID')),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    subtotal NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (subtotal >= 0),
    tax_total NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (tax_total >= 0),
    total NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (total >= 0),
    amount_paid NUMERIC(20,2) NOT NULL DEFAULT 0 CHECK (amount_paid >= 0 AND amount_paid <= total),
    issued_at TIMESTAMP(0) WITHOUT TIME ZONE,
    due_date DATE NOT NULL,
    voided_at TIMESTAMP(0) WITHOUT TIME ZONE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.sales_invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES public.sales_invoices(id) ON DELETE RESTRICT,
    sales_order_item_id UUID REFERENCES public.sales_order_items(id) ON DELETE RESTRICT,
    line_number INTEGER NOT NULL CHECK (line_number > 0),
    description TEXT NOT NULL,
    quantity NUMERIC(20,4) NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(20,2) NOT NULL CHECK (unit_price >= 0),
    tax_rate NUMERIC(7,4) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
    line_total NUMERIC(20,2) NOT NULL CHECK (line_total >= 0),
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (invoice_id, line_number)
);

CREATE TABLE IF NOT EXISTS public.sales_invoice_payment_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES public.sales_invoices(id) ON DELETE RESTRICT,
    payment_id UUID NOT NULL REFERENCES public.payments(id) ON DELETE RESTRICT,
    amount NUMERIC(20,2) NOT NULL CHECK (amount > 0),
    allocated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (invoice_id, payment_id)
);

CREATE TABLE IF NOT EXISTS public.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(60) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    unit VARCHAR(20) NOT NULL DEFAULT 'UNIT',
    is_stocked BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'sales_order_items_product_id_fk'
          AND conrelid = 'public.sales_order_items'::regclass
    ) THEN
        ALTER TABLE public.sales_order_items
            ADD CONSTRAINT sales_order_items_product_id_fk
            FOREIGN KEY (product_id) REFERENCES public.products(id) ON DELETE RESTRICT;
    END IF;
END;
$$;

CREATE TABLE IF NOT EXISTS public.warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    address JSONB NOT NULL DEFAULT '{}'::JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.inventory_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES public.products(id) ON DELETE RESTRICT,
    warehouse_id UUID NOT NULL REFERENCES public.warehouses(id) ON DELETE RESTRICT,
    on_hand NUMERIC(20,4) NOT NULL DEFAULT 0 CHECK (on_hand >= 0),
    reserved NUMERIC(20,4) NOT NULL DEFAULT 0 CHECK (reserved >= 0 AND reserved <= on_hand),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (product_id, warehouse_id)
);

CREATE TABLE IF NOT EXISTS public.inventory_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_order_item_id UUID NOT NULL REFERENCES public.sales_order_items(id) ON DELETE RESTRICT,
    product_id UUID NOT NULL REFERENCES public.products(id) ON DELETE RESTRICT,
    warehouse_id UUID NOT NULL REFERENCES public.warehouses(id) ON DELETE RESTRICT,
    quantity NUMERIC(20,4) NOT NULL CHECK (quantity > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'RESERVED' CHECK (status IN ('RESERVED', 'RELEASED', 'FULFILLED')),
    reserved_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    released_at TIMESTAMP(0) WITHOUT TIME ZONE,
    fulfilled_at TIMESTAMP(0) WITHOUT TIME ZONE
);

CREATE TABLE IF NOT EXISTS public.delivery_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_number VARCHAR(50) NOT NULL UNIQUE DEFAULT public.next_document_number('delivery'),
    sales_order_id UUID NOT NULL REFERENCES public.sales_orders(id) ON DELETE RESTRICT,
    warehouse_id UUID NOT NULL REFERENCES public.warehouses(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PROCESSED', 'CANCELLED')),
    recipient_name VARCHAR(150),
    delivery_address JSONB NOT NULL DEFAULT '{}'::JSONB,
    notes TEXT,
    processed_at TIMESTAMP(0) WITHOUT TIME ZONE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.delivery_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_order_id UUID NOT NULL REFERENCES public.delivery_orders(id) ON DELETE RESTRICT,
    sales_order_item_id UUID NOT NULL REFERENCES public.sales_order_items(id) ON DELETE RESTRICT,
    quantity NUMERIC(20,4) NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (delivery_order_id, sales_order_item_id)
);

CREATE TABLE IF NOT EXISTS public.inventory_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES public.products(id) ON DELETE RESTRICT,
    warehouse_id UUID NOT NULL REFERENCES public.warehouses(id) ON DELETE RESTRICT,
    movement_type VARCHAR(20) NOT NULL CHECK (movement_type IN ('OPENING', 'RECEIPT', 'DELIVERY', 'ADJUSTMENT')),
    quantity_delta NUMERIC(20,4) NOT NULL CHECK (quantity_delta <> 0),
    reference_type VARCHAR(40) NOT NULL,
    reference_id UUID NOT NULL,
    occurred_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (movement_type, reference_type, reference_id, product_id, warehouse_id)
);

CREATE TABLE IF NOT EXISTS public.sales_activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(40) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(60) NOT NULL,
    actor_id UUID,
    actor_name VARCHAR(150),
    old_values JSONB,
    new_values JSONB,
    occurred_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION public.validate_financial_transaction_balance()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_transaction_id UUID;
    transaction_status VARCHAR(20);
    debit_total NUMERIC(20,2);
    credit_total NUMERIC(20,2);
    entry_count BIGINT;
BEGIN
        IF TG_TABLE_NAME = 'financial_transactions' THEN
            target_transaction_id := NEW.id;
        ELSIF TG_OP = 'DELETE' THEN
            target_transaction_id := OLD.financial_transaction_id;
        ELSE
            target_transaction_id := NEW.financial_transaction_id;
        END IF;

    SELECT status INTO transaction_status
    FROM public.financial_transactions
    WHERE id = target_transaction_id;

    IF transaction_status NOT IN ('POSTED', 'COMPLETED') THEN
        RETURN NULL;
    END IF;

    SELECT COALESCE(SUM(debit), 0), COALESCE(SUM(credit), 0), COUNT(*)
    INTO debit_total, credit_total, entry_count
    FROM public.ledger_entries
    WHERE financial_transaction_id = target_transaction_id;

    IF entry_count < 2 OR debit_total <> credit_total OR debit_total <= 0 THEN
        RAISE EXCEPTION 'posted financial transaction % must have balanced ledger entries', target_transaction_id;
    END IF;

    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS financial_transactions_balance_check ON public.financial_transactions;
CREATE CONSTRAINT TRIGGER financial_transactions_balance_check
    AFTER INSERT OR UPDATE OF status ON public.financial_transactions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION public.validate_financial_transaction_balance();

DROP TRIGGER IF EXISTS ledger_entries_balance_check ON public.ledger_entries;
CREATE CONSTRAINT TRIGGER ledger_entries_balance_check
    AFTER INSERT OR UPDATE OR DELETE ON public.ledger_entries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION public.validate_financial_transaction_balance();

CREATE OR REPLACE FUNCTION public.prevent_posted_financial_transaction_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.status IN ('POSTED', 'COMPLETED') THEN
        RAISE EXCEPTION 'posted financial transactions are immutable';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS financial_transactions_immutable ON public.financial_transactions;
CREATE TRIGGER financial_transactions_immutable
    BEFORE UPDATE OR DELETE ON public.financial_transactions
    FOR EACH ROW EXECUTE FUNCTION public.prevent_posted_financial_transaction_mutation();

CREATE OR REPLACE FUNCTION public.prevent_posted_ledger_entry_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    transaction_status VARCHAR(20);
BEGIN
    SELECT status INTO transaction_status
    FROM public.financial_transactions
    WHERE id = OLD.financial_transaction_id;

    IF transaction_status IN ('POSTED', 'COMPLETED') THEN
        RAISE EXCEPTION 'ledger entries for posted financial transactions are immutable';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS ledger_entries_immutable ON public.ledger_entries;
CREATE TRIGGER ledger_entries_immutable
    BEFORE UPDATE OR DELETE ON public.ledger_entries
    FOR EACH ROW EXECUTE FUNCTION public.prevent_posted_ledger_entry_mutation();

CREATE OR REPLACE FUNCTION public.prevent_inventory_movement_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'inventory movements are immutable';
END;
$$;

DROP TRIGGER IF EXISTS inventory_movements_immutable ON public.inventory_movements;
CREATE TRIGGER inventory_movements_immutable
    BEFORE UPDATE OR DELETE ON public.inventory_movements
    FOR EACH ROW EXECUTE FUNCTION public.prevent_inventory_movement_mutation();

CREATE INDEX IF NOT EXISTS idx_sales_rfqs_customer_received
    ON public.sales_rfqs (customer_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_sales_quotations_customer_status
    ON public.sales_quotations (customer_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sales_orders_customer_status
    ON public.sales_orders (customer_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_proforma_invoices_order
    ON public.proforma_invoices (sales_order_id, status);
CREATE INDEX IF NOT EXISTS idx_sales_invoices_customer_due
    ON public.sales_invoices (customer_id, due_date, status);
CREATE INDEX IF NOT EXISTS idx_sales_invoice_allocations_payment
    ON public.sales_invoice_payment_allocations (payment_id);
CREATE INDEX IF NOT EXISTS idx_inventory_balances_available
    ON public.inventory_balances (product_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inventory_reservations_order_status
    ON public.inventory_reservations (sales_order_item_id, status);
CREATE INDEX IF NOT EXISTS idx_delivery_orders_order_status
    ON public.delivery_orders (sales_order_id, status);
CREATE INDEX IF NOT EXISTS idx_inventory_movements_product_time
    ON public.inventory_movements (product_id, warehouse_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_sales_activity_entity_time
    ON public.sales_activity_logs (entity_type, entity_id, occurred_at DESC);

CREATE OR REPLACE VIEW public.sales_invoice_balances AS
SELECT
    invoice.id AS invoice_id,
    invoice.invoice_number,
    invoice.customer_id,
    invoice.status AS document_status,
    CASE
        WHEN invoice.status IN ('VOID', 'DRAFT') THEN invoice.status
        WHEN COALESCE(SUM(allocation.amount), 0) >= invoice.total THEN 'PAID'
        WHEN COALESCE(SUM(allocation.amount), 0) > 0 THEN 'PARTIALLY_PAID'
        WHEN invoice.due_date < CURRENT_DATE THEN 'OVERDUE'
        ELSE 'ISSUED'
    END AS payment_status,
    invoice.currency,
    invoice.total,
    COALESCE(SUM(allocation.amount), 0)::NUMERIC(20,2) AS allocated_amount,
    GREATEST(invoice.total - COALESCE(SUM(allocation.amount), 0), 0)::NUMERIC(20,2) AS balance_due,
    invoice.due_date
FROM public.sales_invoices AS invoice
LEFT JOIN public.sales_invoice_payment_allocations AS allocation
    ON allocation.invoice_id = invoice.id
GROUP BY invoice.id;

COMMIT;