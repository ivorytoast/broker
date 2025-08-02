package main

import (
	"context"
	"errors"
	"fmt"
	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
	"log"
	"time"
)

var (
	polygonClient *polygon.Client
)

func InitPolygonClient(apiKey string) {
	polygonApiKey = apiKey
	polygonClient = polygon.New(apiKey)
}

func runPolygonRequest(symbol string) (*models.GetDailyOpenCloseAggResponse, error) {
	if polygonClient == nil {
		return nil, fmt.Errorf("polygon client not initialized")
	}

	on := time.Now().AddDate(0, 0, -2)

	params := models.GetDailyOpenCloseAggParams{
		Ticker: symbol,
		Date:   models.Date(on),
	}.WithAdjusted(true)

	res, err := polygonClient.GetDailyOpenCloseAgg(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		var errResp *models.ErrorResponse
		if errors.As(err, &errResp) {
			if errResp.StatusCode == 404 {
				log.Print("Symbol not found")
				return nil, err
			}
			log.Printf("Unknown status code [%v] from Polygon", errResp.StatusCode)
			return nil, err
		}
		log.Print("Not a Polygon error")
		return nil, err
	}

	return res, nil
}
