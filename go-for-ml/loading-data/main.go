package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

type Customer struct {
	ID       int
	Name     string
	Email    string
	City     string
	Purchase float64
}

func readCustomers(filename string) ([]Customer, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	customers := []Customer{}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		customer := Customer{
			ID:       mustAtoi(record[0]),
			Name:     record[1],
			Email:    record[2],
			City:     record[3],
			Purchase: mustParseFloat(record[4]),
		}
		customers = append(customers, customer)
	}
	return customers, nil
}

func mustParseFloat(s string) float64 {
	return float64(mustAtoi(s))
}

func mustAtoi(s string) int {
	result, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return result
}

type Product struct {
	ID    int
	Name  string
	Price float64
}

func getProducts(db *sql.DB) ([]Product, error) {
	rows, err := db.Query("SELECT id, name, price FROM products")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := []Product{}
	for rows.Next() {
		var product Product
		err := rows.Scan(&product.ID, &product.Name, &product.Price)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

type WeatherData struct {
	Temperature float64
	Humidity    int
	Conditions  string
}

func getWeather(city string) (*WeatherData, error) {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=YOUR_API_KEY", city)
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var data WeatherData
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func highSpenders(customers []Customer) []Customer {
	var spenders []Customer
	for _, customer := range customers {
		if customer.Purchase > 100 {
			spenders = append(spenders, customer)
		}
	}
	return spenders
}

func normalizePurchases(customers []Customer) []Customer {
	maxPurchase := 0.0
	for _, customer := range customers {
		if customer.Purchase > maxPurchase {
			maxPurchase = customer.Purchase
		}
	}

	for i := range customers {
		customers[i].Purchase /= maxPurchase
	}
	return customers
}
