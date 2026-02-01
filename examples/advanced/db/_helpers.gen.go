package advanceddb

import (
	"database/sql"

	"github.com/electwix/db-catalyst/examples/advanced/types"
)

type CreateOrderRow struct {
	Id          int32
	UserId      int32
	Status      int32
	TotalAmount int32
	CreatedAt   types.Money
	UpdatedAt   *types.Money
}

func scanCreateOrderRow(rows sql.Rows) (CreateOrderRow, error) {
	var item CreateOrderRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Status, &item.TotalAmount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreateProductRow struct {
	Id        int32
	Sku       int32
	Name      types.SKU
	Price     int32
	CreatedAt types.Money
}

func scanCreateProductRow(rows sql.Rows) (CreateProductRow, error) {
	var item CreateProductRow
	if err := rows.Scan(&item.Id, &item.Sku, &item.Name, &item.Price, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type CreateUserRow struct {
	Id        int32
	Email     int32
	CreatedAt types.Money
}

func scanCreateUserRow(rows sql.Rows) (CreateUserRow, error) {
	var item CreateUserRow
	if err := rows.Scan(&item.Id, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetOrderRow struct {
	Id          int32
	UserId      int32
	Status      int32
	TotalAmount int32
	CreatedAt   types.Money
	UpdatedAt   *types.Money
}

func scanGetOrderRow(rows sql.Rows) (GetOrderRow, error) {
	var item GetOrderRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Status, &item.TotalAmount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetOrderStatisticsRow struct {
	TotalOrders  int32
	TotalRevenue *int32
}

func scanGetOrderStatisticsRow(rows sql.Rows) (GetOrderStatisticsRow, error) {
	var item GetOrderStatisticsRow
	if err := rows.Scan(&item.TotalOrders, &item.TotalRevenue); err != nil {
		return item, err
	}
	return item, nil
}

type GetOrdersByStatusRow struct {
	Id          int32
	UserId      int32
	Status      int32
	TotalAmount int32
	CreatedAt   types.Money
}

func scanGetOrdersByStatusRow(rows sql.Rows) (GetOrdersByStatusRow, error) {
	var item GetOrdersByStatusRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Status, &item.TotalAmount, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetProductBySkuRow struct {
	Id        int32
	Sku       int32
	Name      types.SKU
	Price     int32
	CreatedAt types.Money
}

func scanGetProductBySkuRow(rows sql.Rows) (GetProductBySkuRow, error) {
	var item GetProductBySkuRow
	if err := rows.Scan(&item.Id, &item.Sku, &item.Name, &item.Price, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetProductRow struct {
	Id        int32
	Sku       int32
	Name      types.SKU
	Price     int32
	CreatedAt types.Money
}

func scanGetProductRow(rows sql.Rows) (GetProductRow, error) {
	var item GetProductRow
	if err := rows.Scan(&item.Id, &item.Sku, &item.Name, &item.Price, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserByEmailRow struct {
	Id        int32
	Email     int32
	CreatedAt types.Money
}

func scanGetUserByEmailRow(rows sql.Rows) (GetUserByEmailRow, error) {
	var item GetUserByEmailRow
	if err := rows.Scan(&item.Id, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type GetUserRow struct {
	Id        int32
	Email     int32
	CreatedAt types.Money
}

func scanGetUserRow(rows sql.Rows) (GetUserRow, error) {
	var item GetUserRow
	if err := rows.Scan(&item.Id, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListOrdersByUserRow struct {
	Id          int32
	Status      int32
	TotalAmount int32
	CreatedAt   types.Money
}

func scanListOrdersByUserRow(rows sql.Rows) (ListOrdersByUserRow, error) {
	var item ListOrdersByUserRow
	if err := rows.Scan(&item.Id, &item.Status, &item.TotalAmount, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListProductsRow struct {
	Id        int32
	Sku       int32
	Name      types.SKU
	Price     int32
	CreatedAt types.Money
}

func scanListProductsRow(rows sql.Rows) (ListProductsRow, error) {
	var item ListProductsRow
	if err := rows.Scan(&item.Id, &item.Sku, &item.Name, &item.Price, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type ListUsersRow struct {
	Id        int32
	Email     int32
	CreatedAt types.Money
}

func scanListUsersRow(rows sql.Rows) (ListUsersRow, error) {
	var item ListUsersRow
	if err := rows.Scan(&item.Id, &item.Email, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdateOrderStatusRow struct {
	Id          int32
	UserId      int32
	Status      int32
	TotalAmount int32
	CreatedAt   types.Money
	UpdatedAt   *types.Money
}

func scanUpdateOrderStatusRow(rows sql.Rows) (UpdateOrderStatusRow, error) {
	var item UpdateOrderStatusRow
	if err := rows.Scan(&item.Id, &item.UserId, &item.Status, &item.TotalAmount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	return item, nil
}

type UpdateProductPriceRow struct {
	Id        int32
	Sku       int32
	Name      types.SKU
	Price     int32
	CreatedAt types.Money
}

func scanUpdateProductPriceRow(rows sql.Rows) (UpdateProductPriceRow, error) {
	var item UpdateProductPriceRow
	if err := rows.Scan(&item.Id, &item.Sku, &item.Name, &item.Price, &item.CreatedAt); err != nil {
		return item, err
	}
	return item, nil
}
