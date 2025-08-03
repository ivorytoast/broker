package stocks

import (
	"broker/engine"
	"context"
	"errors"
	"fmt"
	polygonws "github.com/polygon-io/client-go/websocket"
	wsModels "github.com/polygon-io/client-go/websocket/models"
	"log"
	"time"

	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
)

type PolygonService struct {
	client   *polygon.Client
	wsClient *polygonws.Client
}

func NewPolygonService(apiKey string) *PolygonService {
	polygonWsClient, err := polygonws.New(polygonws.Config{
		APIKey: apiKey,
		Feed:   polygonws.RealTime,
		Market: polygonws.Crypto,
	})
	if err != nil {
		panic(err)
	}
	return &PolygonService{
		client:   polygon.New(apiKey),
		wsClient: polygonWsClient,
	}
}

func (ps *PolygonService) StartWebsocket(e *engine.Engine) {
	if err := ps.wsClient.Connect(); err != nil {
		panic(err)
		return
	}

	if err := ps.wsClient.Subscribe(polygonws.CryptoSecAggs); err != nil {
		log.Fatal(err)
	}

	if err := ps.wsClient.Subscribe(polygonws.CryptoTrades); err != nil {
		log.Fatal(err)
	}

	var allowedPairs = map[string]bool{
		"BTC-USD":    true,
		"ETH-USD":    true,
		"XRP-USD":    true,
		"SOL-USD":    true,
		"OMNI-USD":   true,
		"CRV-USD":    true,
		"USDT-USD":   true,
		"SUI-USD":    true,
		"GNO-USD":    true,
		"LTC-USD":    true,
		"VET-USD":    true,
		"MORPHO-USD": true,
		"XCN-USD":    true,
		"RSR-USD":    true,
		"CVX-USD":    true,
		"ADA-USD":    true,
		"METIS-USD":  true,
		"PEPE-USD":   true,
		"MDT-USD":    true,
	}

	for {
		select {
		case err := <-ps.wsClient.Error():
			log.Fatal(err)
		case out, more := <-ps.wsClient.Output():
			if !more {
				return
			}

			switch out.(type) {
			case wsModels.CryptoTrade:
				trade := out.(wsModels.CryptoTrade)
				if !allowedPairs[trade.Pair] {
					continue
				}
				e.Broadcast(fmt.Sprintf("[crypto][%v,%v]", trade.Pair, trade.Price))
			case wsModels.CurrencyAgg:
				agg := out.(wsModels.CurrencyAgg)
				if !allowedPairs[agg.Pair] {
					continue
				}
				e.Broadcast(fmt.Sprintf("[crypto][%v,%v]", agg.Pair, agg.Close))
			}
		}
	}
}

func (ps *PolygonService) RunRequest(symbol string) (*models.GetDailyOpenCloseAggResponse, error) {
	if ps.client == nil {
		return nil, fmt.Errorf("polygon client not initialized")
	}

	on := time.Now().AddDate(0, 0, -2)

	params := models.GetDailyOpenCloseAggParams{
		Ticker: symbol,
		Date:   models.Date(on),
	}.WithAdjusted(true)

	res, err := ps.client.GetDailyOpenCloseAgg(context.Background(), params)
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
