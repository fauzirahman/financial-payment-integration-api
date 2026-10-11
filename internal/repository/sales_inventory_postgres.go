package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fauzirahman/financial-payment-integration-api/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSalesOrderNotFound = errors.New("sales order not found")
	ErrSalesOrderState    = errors.New("sales order cannot transition from current state")
	ErrInsufficientStock  = errors.New("insufficient available inventory")
	ErrWarehouseNotFound  = errors.New("active warehouse not found")
	ErrProductNotFound    = errors.New("active stocked product not found")
)

type PostgresSalesInventoryRepository struct {
	db *pgxpool.Pool
}

func NewPostgresSalesInventoryRepository(db *pgxpool.Pool) *PostgresSalesInventoryRepository {
	return &PostgresSalesInventoryRepository{db: db}
}

func (r *PostgresSalesInventoryRepository) CreateSalesOrder(ctx context.Context, order *model.SalesOrder) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	address := string(order.DeliveryAddress)
	if address == "" {
		address = "{}"
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO sales_orders (
			customer_id, quotation_id, status, currency, subtotal, tax_total, total,
			payment_terms, delivery_address
		)
		VALUES ($1, NULLIF($2, '')::uuid, 'DRAFT', $3, $4, $5, $6, $7, $8::jsonb)
		RETURNING id::text, order_number, status, created_at, updated_at
	`, order.CustomerID, order.QuotationID, order.Currency, order.Subtotal, order.TaxTotal, order.Total, order.PaymentTerms, address).Scan(
		&order.ID, &order.OrderNumber, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		return err
	}
	for index := range order.Items {
		item := &order.Items[index]
		err = tx.QueryRow(ctx, `
			INSERT INTO sales_order_items (
				sales_order_id, line_number, product_id, sku, description, quantity,
				unit, unit_price, tax_rate, line_total
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id::text
		`, order.ID, index+1, item.ProductID, item.SKU, item.Description, item.Quantity, item.Unit, item.UnitPrice, item.TaxRate, item.LineTotal).Scan(&item.ID)
		if err != nil {
			return err
		}
	}
	if err := insertSalesActivity(ctx, tx, order.ID, "CREATED", nil, map[string]any{"status": "DRAFT", "order_number": order.OrderNumber}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresSalesInventoryRepository) ListSalesOrders(ctx context.Context) ([]model.SalesOrder, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, order_number, customer_id::text, COALESCE(quotation_id::text, ''),
			status, currency, subtotal::float8, tax_total::float8, total::float8,
			COALESCE(payment_terms, ''), delivery_address, created_at, updated_at
		FROM sales_orders
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.SalesOrder, 0)
	for rows.Next() {
		var order model.SalesOrder
		var address []byte
		if err := rows.Scan(&order.ID, &order.OrderNumber, &order.CustomerID, &order.QuotationID, &order.Status,
			&order.Currency, &order.Subtotal, &order.TaxTotal, &order.Total, &order.PaymentTerms,
			&address, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		order.DeliveryAddress = address
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *PostgresSalesInventoryRepository) GetSalesOrder(ctx context.Context, id string) (*model.SalesOrder, error) {
	var order model.SalesOrder
	var address []byte
	err := r.db.QueryRow(ctx, `
		SELECT id::text, order_number, customer_id::text, COALESCE(quotation_id::text, ''),
			status, currency, subtotal::float8, tax_total::float8, total::float8,
			COALESCE(payment_terms, ''), delivery_address, created_at, updated_at
		FROM sales_orders WHERE id = $1
	`, id).Scan(&order.ID, &order.OrderNumber, &order.CustomerID, &order.QuotationID, &order.Status,
		&order.Currency, &order.Subtotal, &order.TaxTotal, &order.Total, &order.PaymentTerms,
		&address, &order.CreatedAt, &order.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSalesOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	order.DeliveryAddress = address
	order.Items, err = readSalesOrderItems(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *PostgresSalesInventoryRepository) ListSalesActivity(ctx context.Context, id string) ([]model.SalesActivity, error) {
	if err := requireSalesOrder(ctx, r.db, id); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, entity_type, entity_id::text, action, COALESCE(actor_name, ''),
			old_values, new_values, occurred_at
		FROM sales_activity_logs
		WHERE entity_type = 'SALES_ORDER' AND entity_id = $1
		ORDER BY occurred_at, id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	activities := make([]model.SalesActivity, 0)
	for rows.Next() {
		var activity model.SalesActivity
		var oldValues, newValues []byte
		if err := rows.Scan(&activity.ID, &activity.EntityType, &activity.EntityID, &activity.Action,
			&activity.ActorName, &oldValues, &newValues, &activity.OccurredAt); err != nil {
			return nil, err
		}
		activity.OldValues = oldValues
		activity.NewValues = newValues
		activities = append(activities, activity)
	}
	return activities, rows.Err()
}

func (r *PostgresSalesInventoryRepository) ListDeliveries(ctx context.Context, id string) ([]model.DeliveryOrder, error) {
	if err := requireSalesOrder(ctx, r.db, id); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, delivery_number, sales_order_id::text, warehouse_id::text, status, processed_at
		FROM delivery_orders WHERE sales_order_id = $1 ORDER BY created_at, id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deliveries := make([]model.DeliveryOrder, 0)
	for rows.Next() {
		var delivery model.DeliveryOrder
		if err := rows.Scan(&delivery.ID, &delivery.DeliveryNumber, &delivery.SalesOrderID, &delivery.WarehouseID,
			&delivery.Status, &delivery.ProcessedAt); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for index := range deliveries {
		deliveries[index].Items, err = readDeliveryItems(ctx, r.db, deliveries[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return deliveries, nil
}

func (r *PostgresSalesInventoryRepository) ConfirmSalesOrder(ctx context.Context, id, warehouseID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM sales_orders WHERE id = $1 FOR UPDATE`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSalesOrderNotFound
	}
	if err != nil {
		return err
	}
	if status != "DRAFT" {
		return ErrSalesOrderState
	}
	if err := requireActiveWarehouse(ctx, tx, warehouseID); err != nil {
		return err
	}
	items, err := readSalesOrderItemsForUpdate(ctx, tx, id)
	if err != nil {
		return err
	}
	for _, item := range items {
		var stocked bool
		err := tx.QueryRow(ctx, `SELECT is_stocked FROM products WHERE id = $1 AND is_active FOR SHARE`, item.ProductID).Scan(&stocked)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && !stocked) {
			return ErrProductNotFound
		}
		if err != nil {
			return err
		}
		var balanceID string
		err = tx.QueryRow(ctx, `
			UPDATE inventory_balances
			SET reserved = reserved + $1, updated_at = NOW()
			WHERE product_id = $2 AND warehouse_id = $3 AND on_hand - reserved >= $1
			RETURNING id::text
		`, item.Quantity, item.ProductID, warehouseID).Scan(&balanceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: product %s", ErrInsufficientStock, item.ProductID)
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_reservations (sales_order_item_id, product_id, warehouse_id, quantity)
			VALUES ($1, $2, $3, $4)
		`, item.ID, item.ProductID, warehouseID, item.Quantity); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sales_orders SET status = 'CONFIRMED', confirmed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id); err != nil {
		return err
	}
	if err := insertSalesActivity(ctx, tx, id, "CONFIRMED", map[string]any{"status": status}, map[string]any{"status": "CONFIRMED", "warehouse_id": warehouseID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresSalesInventoryRepository) DeliverSalesOrder(ctx context.Context, id, warehouseID string) (*model.DeliveryOrder, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM sales_orders WHERE id = $1 FOR UPDATE`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSalesOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != "CONFIRMED" {
		return nil, ErrSalesOrderState
	}
	if err := requireActiveWarehouse(ctx, tx, warehouseID); err != nil {
		return nil, err
	}
	items, err := readSalesOrderItemsForUpdate(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		var reserved float64
		err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(quantity), 0)::float8
			FROM inventory_reservations
			WHERE sales_order_item_id = $1 AND warehouse_id = $2 AND status = 'RESERVED'
		`, item.ID, warehouseID).Scan(&reserved)
		if err != nil {
			return nil, err
		}
		if reserved < item.Quantity {
			return nil, fmt.Errorf("%w: reservation missing for order item %s", ErrInsufficientStock, item.ID)
		}
	}

	var delivery model.DeliveryOrder
	err = tx.QueryRow(ctx, `
		INSERT INTO delivery_orders (sales_order_id, warehouse_id, status, processed_at)
		VALUES ($1, $2, 'PROCESSED', NOW())
		RETURNING id::text, delivery_number, status, processed_at
	`, id, warehouseID).Scan(&delivery.ID, &delivery.DeliveryNumber, &delivery.Status, &delivery.ProcessedAt)
	if err != nil {
		return nil, err
	}
	delivery.SalesOrderID = id
	delivery.WarehouseID = warehouseID
	movementQuantities := make(map[string]float64)
	for _, item := range items {
		result, err := tx.Exec(ctx, `
			UPDATE inventory_balances
			SET on_hand = on_hand - $1, reserved = reserved - $1, updated_at = NOW()
			WHERE product_id = $2 AND warehouse_id = $3 AND on_hand >= $1 AND reserved >= $1
		`, item.Quantity, item.ProductID, warehouseID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() != 1 {
			return nil, ErrInsufficientStock
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO delivery_order_items (delivery_order_id, sales_order_item_id, quantity)
			VALUES ($1, $2, $3)
		`, delivery.ID, item.ID, item.Quantity); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_reservations
			SET status = 'FULFILLED', fulfilled_at = NOW()
			WHERE sales_order_item_id = $1 AND warehouse_id = $2 AND status = 'RESERVED'
		`, item.ID, warehouseID); err != nil {
			return nil, err
		}
		movementQuantities[item.ProductID] += item.Quantity
	}
	for productID, quantity := range movementQuantities {
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_movements (product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id)
			VALUES ($1, $2, 'DELIVERY', -$3, 'DELIVERY_ORDER', $4)
		`, productID, warehouseID, quantity, delivery.ID); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE sales_orders SET status = 'DELIVERED', updated_at = NOW() WHERE id = $1`, id); err != nil {
		return nil, err
	}
	if err := insertSalesActivity(ctx, tx, id, "DELIVERY_PROCESSED", map[string]any{"status": status}, map[string]any{"status": "DELIVERED", "delivery_number": delivery.DeliveryNumber}); err != nil {
		return nil, err
	}
	if err := insertSalesActivity(ctx, tx, delivery.ID, "PROCESSED", nil, map[string]any{"delivery_number": delivery.DeliveryNumber, "sales_order_id": id}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &delivery, nil
}

func (r *PostgresSalesInventoryRepository) ListInventory(ctx context.Context) ([]model.InventoryBalance, error) {
	rows, err := r.db.Query(ctx, `
		SELECT product.id::text, product.sku, product.name,
			warehouse.id::text, warehouse.code || ' - ' || warehouse.name,
			balance.on_hand::float8, balance.reserved::float8,
			(balance.on_hand - balance.reserved)::float8
		FROM inventory_balances AS balance
		JOIN products AS product ON product.id = balance.product_id
		JOIN warehouses AS warehouse ON warehouse.id = balance.warehouse_id
		ORDER BY product.sku, warehouse.code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	balances := make([]model.InventoryBalance, 0)
	for rows.Next() {
		var balance model.InventoryBalance
		if err := rows.Scan(&balance.ProductID, &balance.SKU, &balance.ProductName, &balance.WarehouseID,
			&balance.Warehouse, &balance.OnHand, &balance.Reserved, &balance.Available); err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}
	return balances, rows.Err()
}

func (r *PostgresSalesInventoryRepository) ListInventoryMovements(ctx context.Context, productID, warehouseID string, limit int) ([]model.InventoryMovement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT movement.id::text, product.id::text, product.sku, product.name,
			warehouse.id::text, warehouse.code || ' - ' || warehouse.name,
			movement.movement_type, movement.quantity_delta::float8,
			movement.reference_type, movement.reference_id::text, movement.occurred_at
		FROM inventory_movements AS movement
		JOIN products AS product ON product.id = movement.product_id
		JOIN warehouses AS warehouse ON warehouse.id = movement.warehouse_id
		WHERE ($1 = '' OR movement.product_id::text = $1)
		  AND ($2 = '' OR movement.warehouse_id::text = $2)
		ORDER BY movement.occurred_at DESC, movement.id DESC
		LIMIT $3
	`, productID, warehouseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	movements := make([]model.InventoryMovement, 0)
	for rows.Next() {
		var movement model.InventoryMovement
		if err := rows.Scan(&movement.ID, &movement.ProductID, &movement.SKU, &movement.ProductName,
			&movement.WarehouseID, &movement.Warehouse, &movement.MovementType, &movement.QuantityDelta,
			&movement.ReferenceType, &movement.ReferenceID, &movement.OccurredAt); err != nil {
			return nil, err
		}
		movements = append(movements, movement)
	}
	return movements, rows.Err()
}

func (r *PostgresSalesInventoryRepository) RecordInventoryMovement(ctx context.Context, movementType string, request *model.InventoryMovementRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := requireActiveWarehouse(ctx, tx, request.WarehouseID); err != nil {
		return err
	}
	var stocked bool
	err = tx.QueryRow(ctx, `SELECT is_stocked FROM products WHERE id = $1 AND is_active`, request.ProductID).Scan(&stocked)
	if errors.Is(err, pgx.ErrNoRows) || !stocked {
		return ErrProductNotFound
	}
	if err != nil {
		return err
	}
	if request.QuantityDelta < 0 {
		var balanceID string
		err = tx.QueryRow(ctx, `
			UPDATE inventory_balances
			SET on_hand = on_hand + $1, updated_at = NOW()
			WHERE product_id = $2 AND warehouse_id = $3
			  AND on_hand + $1 >= reserved AND on_hand + $1 >= 0
			RETURNING id::text
		`, request.QuantityDelta, request.ProductID, request.WarehouseID).Scan(&balanceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInsufficientStock
		}
		if err != nil {
			return err
		}
	} else {
		var balanceID string
		err = tx.QueryRow(ctx, `
			INSERT INTO inventory_balances (product_id, warehouse_id, on_hand, reserved)
			VALUES ($1, $2, $3, 0)
			ON CONFLICT (product_id, warehouse_id) DO UPDATE
			SET on_hand = inventory_balances.on_hand + EXCLUDED.on_hand,
				updated_at = NOW()
			WHERE inventory_balances.on_hand + EXCLUDED.on_hand >= inventory_balances.reserved
			RETURNING id::text
		`, request.ProductID, request.WarehouseID, request.QuantityDelta).Scan(&balanceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInsufficientStock
		}
		if err != nil {
			return err
		}
	}
	var movementID string
	err = tx.QueryRow(ctx, `
		INSERT INTO inventory_movements (product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id)
		VALUES ($1, $2, $3, $4, 'MANUAL', gen_random_uuid())
		RETURNING id::text
	`, request.ProductID, request.WarehouseID, movementType, request.QuantityDelta).Scan(&movementID)
	if err != nil {
		return err
	}
	if err := insertInventoryActivity(ctx, tx, movementID, movementType, request); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresSalesInventoryRepository) CreateProduct(ctx context.Context, product *model.Product) error {
	if err := r.db.QueryRow(ctx, `
		INSERT INTO products (sku, name, description, unit, is_stocked, is_active)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6)
		RETURNING id::text, created_at
	`, product.SKU, product.Name, product.Description, product.Unit, product.IsStocked, product.IsActive).Scan(&product.ID, &product.CreatedAt); err != nil {
		return err
	}
	return nil
}

func (r *PostgresSalesInventoryRepository) ListProducts(ctx context.Context) ([]model.Product, error) {
	rows, err := r.db.Query(ctx, `SELECT id::text, sku, name, COALESCE(description, ''), unit, is_stocked, is_active, created_at FROM products ORDER BY sku`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := make([]model.Product, 0)
	for rows.Next() {
		var product model.Product
		if err := rows.Scan(&product.ID, &product.SKU, &product.Name, &product.Description, &product.Unit, &product.IsStocked, &product.IsActive, &product.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

func (r *PostgresSalesInventoryRepository) CreateWarehouse(ctx context.Context, warehouse *model.Warehouse) error {
	address := string(warehouse.Address)
	if address == "" {
		address = "{}"
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, address, is_active)
		VALUES ($1, $2, $3::jsonb, $4)
		RETURNING id::text, created_at
	`, warehouse.Code, warehouse.Name, address, warehouse.IsActive).Scan(&warehouse.ID, &warehouse.CreatedAt)
}

func (r *PostgresSalesInventoryRepository) ListWarehouses(ctx context.Context) ([]model.Warehouse, error) {
	rows, err := r.db.Query(ctx, `SELECT id::text, code, name, address, is_active, created_at FROM warehouses ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	warehouses := make([]model.Warehouse, 0)
	for rows.Next() {
		var warehouse model.Warehouse
		var address []byte
		if err := rows.Scan(&warehouse.ID, &warehouse.Code, &warehouse.Name, &address, &warehouse.IsActive, &warehouse.CreatedAt); err != nil {
			return nil, err
		}
		warehouse.Address = address
		warehouses = append(warehouses, warehouse)
	}
	return warehouses, rows.Err()
}

type salesOrderItemRow struct {
	ID          string
	ProductID   string
	SKU         string
	Description string
	Quantity    float64
	Unit        string
	UnitPrice   float64
	TaxRate     float64
	LineTotal   float64
}

func readSalesOrderItems(ctx context.Context, db *pgxpool.Pool, orderID string) ([]model.SalesOrderItem, error) {
	rows, err := db.Query(ctx, salesOrderItemsQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSalesOrderItems(rows)
}

func readDeliveryItems(ctx context.Context, db *pgxpool.Pool, deliveryID string) ([]model.DeliveryOrderItem, error) {
	rows, err := db.Query(ctx, `
		SELECT delivery_item.sales_order_item_id::text, COALESCE(order_item.sku, ''),
			order_item.description, delivery_item.quantity::float8
		FROM delivery_order_items AS delivery_item
		JOIN sales_order_items AS order_item ON order_item.id = delivery_item.sales_order_item_id
		WHERE delivery_item.delivery_order_id = $1
		ORDER BY order_item.line_number
	`, deliveryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.DeliveryOrderItem, 0)
	for rows.Next() {
		var item model.DeliveryOrderItem
		if err := rows.Scan(&item.SalesOrderItemID, &item.SKU, &item.Description, &item.Quantity); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func readSalesOrderItemsForUpdate(ctx context.Context, tx pgx.Tx, orderID string) ([]salesOrderItemRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, product_id::text, quantity::float8
		FROM sales_order_items WHERE sales_order_id = $1 ORDER BY product_id, id FOR UPDATE
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]salesOrderItemRow, 0)
	for rows.Next() {
		var item salesOrderItemRow
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const salesOrderItemsQuery = `
	SELECT id::text, product_id::text, COALESCE(sku, ''), description,
		quantity::float8, unit, unit_price::float8, tax_rate::float8, line_total::float8
	FROM sales_order_items WHERE sales_order_id = $1 ORDER BY line_number
`

func collectSalesOrderItems(rows pgx.Rows) ([]model.SalesOrderItem, error) {
	items := make([]model.SalesOrderItem, 0)
	for rows.Next() {
		var item model.SalesOrderItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.SKU, &item.Description, &item.Quantity,
			&item.Unit, &item.UnitPrice, &item.TaxRate, &item.LineTotal); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func requireActiveWarehouse(ctx context.Context, tx pgx.Tx, warehouseID string) error {
	var active bool
	err := tx.QueryRow(ctx, `SELECT is_active FROM warehouses WHERE id = $1 FOR SHARE`, warehouseID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) || !active {
		return ErrWarehouseNotFound
	}
	return err
}

func requireSalesOrder(ctx context.Context, db *pgxpool.Pool, orderID string) error {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sales_orders WHERE id = $1)`, orderID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrSalesOrderNotFound
	}
	return nil
}

func insertSalesActivity(ctx context.Context, tx pgx.Tx, entityID, action string, oldValues, newValues any) error {
	oldJSON, err := json.Marshal(oldValues)
	if err != nil {
		return err
	}
	newJSON, err := json.Marshal(newValues)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO sales_activity_logs (entity_type, entity_id, action, old_values, new_values)
		VALUES ('SALES_ORDER', $1, $2, NULLIF($3, 'null')::jsonb, $4::jsonb)
	`, entityID, action, string(oldJSON), string(newJSON))
	return err
}

func insertInventoryActivity(ctx context.Context, tx pgx.Tx, entityID, movementType string, request *model.InventoryMovementRequest) error {
	values, err := json.Marshal(map[string]any{
		"movement_type":  movementType,
		"product_id":     request.ProductID,
		"warehouse_id":   request.WarehouseID,
		"quantity_delta": request.QuantityDelta,
		"reason":         request.Reason,
	})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO sales_activity_logs (entity_type, entity_id, action, new_values)
		VALUES ('INVENTORY_MOVEMENT', $1, $2, $3::jsonb)
	`, entityID, movementType, string(values))
	return err
}
