package plugins

import (
	"fmt"
	"log"
	"math"

	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/model"
	"github.com/stellar/kelp/support/utils"
)

type makerModeFilter struct {
	tradingPair  *model.TradingPair
	exchangeShim api.ExchangeShim
	sdex         *SDEX
}

// MakeFilterMakerMode makes a submit filter based on the passed in submitMode
func MakeFilterMakerMode(submitMode api.SubmitMode, exchangeShim api.ExchangeShim, sdex *SDEX, tradingPair *model.TradingPair) SubmitFilter {
	if submitMode == api.SubmitModeMakerOnly {
		return &makerModeFilter{
			tradingPair:  tradingPair,
			exchangeShim: exchangeShim,
			sdex:         sdex,
		}
	}
	return nil
}

var _ SubmitFilter = &makerModeFilter{}

func (f *makerModeFilter) Apply(ops []build.TransactionMutator, sellingOffers []horizon.Offer, buyingOffers []horizon.Offer) ([]build.TransactionMutator, error) {
	ob, e := f.exchangeShim.GetOrderBook(f.tradingPair, 50)
	if e != nil {
		return nil, fmt.Errorf("could not fetch orderbook: %s", e)
	}

	ops, e = f.filterOps(ops, ob, sellingOffers, buyingOffers)
	if e != nil {
		return nil, fmt.Errorf("could not apply filter: %s", e)
	}
	return ops, nil
}

func isNewLevel(lastPrice *model.Number, priceNumber *model.Number, isSell bool) bool {
	if lastPrice == nil {
		return true
	} else if isSell && priceNumber.AsFloat() > lastPrice.AsFloat() {
		return true
	} else if !isSell && priceNumber.AsFloat() < lastPrice.AsFloat() {
		return true
	}
	return false
}

func (f *makerModeFilter) collateOffers(traderOffers []horizon.Offer, isSell bool) ([]api.Level, error) {
	oc := f.exchangeShim.GetOrderConstraints(f.tradingPair)

	levels := []api.Level{}
	var lastPrice *model.Number
	for _, tOffer := range traderOffers {
		price := float64(tOffer.PriceR.N) / float64(tOffer.PriceR.D)
		amount := utils.AmountStringAsFloat(tOffer.Amount)
		if !isSell {
			price = 1 / price
			amount = amount / price
		}
		priceNumber := model.NumberFromFloat(price, oc.PricePrecision)
		amountNumber := model.NumberFromFloat(amount, oc.VolumePrecision)

		if isNewLevel(lastPrice, priceNumber, isSell) {
			lastPrice = priceNumber
			levels = append(levels, api.Level{
				Price:  *priceNumber,
				Amount: *amountNumber,
			})
		} else if priceNumber.AsFloat() == lastPrice.AsFloat() {
			levels[len(levels)-1].Amount = *levels[len(levels)-1].Amount.Add(*amountNumber)
		} else {
			return nil, fmt.Errorf("invalid ordering of prices (isSell=%v), lastPrice=%s, priceNumber=%s", isSell, lastPrice.AsString(), priceNumber.AsString())
		}
	}
	return levels, nil
}

func (f *makerModeFilter) topOrderPriceExcludingTrader(obSide []model.Order, traderOffers []horizon.Offer, isSell bool) (*model.Number, error) {
	traderLevels, e := f.collateOffers(traderOffers, isSell)
	if e != nil {
		return nil, fmt.Errorf("unable to collate offers: %s", e)
	}

	for i, obOrder := range obSide {
		if i >= len(traderLevels) {
			return obOrder.Price, nil
		}

		traderLevel := traderLevels[i]
		if isNewLevel(obOrder.Price, &traderLevel.Price, isSell) {
			return obOrder.Price, nil
		} else if traderLevel.Amount.AsFloat() < obOrder.Volume.AsFloat() {
			return obOrder.Price, nil
		}
	}

	// orderbook only had trader's orders
	return nil, nil
}

func (f *makerModeFilter) filterOps(
	ops []build.TransactionMutator,
	ob *model.OrderBook,
	sellingOffers []horizon.Offer,
	buyingOffers []horizon.Offer,
) ([]build.TransactionMutator, error) {
	baseAsset, quoteAsset, e := f.sdex.Assets()
	if e != nil {
		return nil, fmt.Errorf("could not get assets: %s", e)
	}

	topBidPrice, e := f.topOrderPriceExcludingTrader(ob.Bids(), buyingOffers, false)
	if e != nil {
		return nil, fmt.Errorf("could not get topOrderPriceExcludingTrader for bids: %s", e)
	}
	topAskPrice, e := f.topOrderPriceExcludingTrader(ob.Asks(), sellingOffers, true)
	if e != nil {
		return nil, fmt.Errorf("could not get topOrderPriceExcludingTrader for asks: %s", e)
	}

	numKeep := 0
	numDropped := 0
	numTransformed := 0
	filteredOps := []build.TransactionMutator{}
	for _, op := range ops {
		var newOp build.TransactionMutator
		var keep bool
		switch o := op.(type) {
		case *build.ManageOfferBuilder:
			newOp, keep, e = f.transformOfferMakerMode(baseAsset, quoteAsset, topBidPrice, topAskPrice, o)
			if e != nil {
				return nil, fmt.Errorf("could not transform offer (pointer case): %s", e)
			}
		case build.ManageOfferBuilder:
			newOp, keep, e = f.transformOfferMakerMode(baseAsset, quoteAsset, topBidPrice, topAskPrice, &o)
			if e != nil {
				return nil, fmt.Errorf("could not check transform offer (non-pointer case): %s", e)
			}
		default:
			newOp = o
			keep = true
		}

		isNewOpNil := newOp == nil || fmt.Sprintf("%v", newOp) == "<nil>"
		if keep {
			if isNewOpNil {
				return nil, fmt.Errorf("we want to keep op but newOp was nil (programmer error?)")
			}
			filteredOps = append(filteredOps, newOp)
			numKeep++
		} else {
			if !isNewOpNil {
				// newOp can be a transformed op to change the op to an effectively "dropped" state
				filteredOps = append(filteredOps, newOp)
				numTransformed++
			} else {
				numDropped++
			}
		}
	}
	log.Printf("makerModeFilter: dropped %d, transformed %d, kept %d ops from original %d ops, len(filteredOps) = %d\n", numDropped, numTransformed, numKeep, len(ops), len(filteredOps))
	return filteredOps, nil
}

func (f *makerModeFilter) transformOfferMakerMode(
	baseAsset horizon.Asset,
	quoteAsset horizon.Asset,
	topBidPrice *model.Number,
	topAskPrice *model.Number,
	op *build.ManageOfferBuilder,
) (*build.ManageOfferBuilder, bool, error) {
	// delete operations should never be dropped
	if op.MO.Amount == 0 {
		return op, true, nil
	}

	isSell, e := utils.IsSelling(baseAsset, quoteAsset, op.MO.Selling, op.MO.Buying)
	if e != nil {
		return nil, false, fmt.Errorf("error when running the isSelling check: %s", e)
	}

	sellPrice := float64(op.MO.Price.N) / float64(op.MO.Price.D)
	var keep bool
	if !isSell && topAskPrice != nil {
		// invert price when buying
		keep = 1/sellPrice < topAskPrice.AsFloat()
		log.Printf("makerModeFilter:  buying, keep = (op price) %.7f < %.7f (topAskPrice): keep = %v", 1/sellPrice, topAskPrice.AsFloat(), keep)
	} else if isSell && topBidPrice != nil {
		keep = sellPrice > topBidPrice.AsFloat()
		log.Printf("makerModeFilter: selling, keep = (op price) %.7f > %.7f (topBidPrice): keep = %v", sellPrice, topBidPrice.AsFloat(), keep)
	} else {
		price := sellPrice
		action := "selling"
		if !isSell {
			price = 1 / price
			action = " buying"
		}
		keep = true
		log.Printf("makerModeFilter: %s, no market (op price = %.7f): keep = %v", action, price, keep)
	}

	if keep {
		return op, true, nil
	}

	// figure out how to convert the offer to a dropped state
	if op.MO.OfferId == 0 {
		// new offers can be dropped
		return nil, false, nil
	} else if op.MO.Amount != 0 {
		// modify offers should be converted to delete offers
		opCopy := *op
		opCopy.MO.Amount = 0
		return &opCopy, false, nil
	}
	return nil, keep, fmt.Errorf("unable to transform manageOffer operation: offerID=%d, amount=%.7f, price=%.7f", op.MO.OfferId, float64(op.MO.Amount)/math.Pow(10, 7), sellPrice)
}
