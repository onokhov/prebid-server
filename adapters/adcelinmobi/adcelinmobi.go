package adcelinmobi

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"net/url"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
)

type AdcelInmobiAdapter struct {
	URI string
}

func (adapter *AdcelInmobiAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	errors := make([]error, 0, len(request.Imp))

	if request.Imp == nil || len(request.Imp) != 1 {
		errors = append(errors, &errortypes.BadInput{
			Message: "Invalid BidRequest. Single imp object required",
		})
		return nil, errors
	}

	imp := request.Imp[0]
	if imp.Video == nil {
		errors = append(errors, &errortypes.BadInput{
			Message: "Only video imp requests allowed",
		})
		return nil, errors
	}
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: fmt.Sprintf("Error while decoding extImpBidder, err: %s.", err),
		})
		return nil, errors
	}
	impExt := openrtb_ext.ExtImpAdcelInmobi{}
	err := json.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: fmt.Sprintf("Error while decoding impExt, err: %s.", err),
		})
		return nil, errors
	}
	if impExt.Plc == "" {
		errors = append(errors, &errortypes.BadInput{
			Message: "No plc present.",
		})
		return nil, errors
	}

	// curl -s -v 'http://api.w.inmobi.com/showad/v3/vast?
	// plid=1610532934699
	// ua=Mozilla%2F5.0%20(iPhone%3B%20CPU%20iPhone%20OS%208_2%20like%20Mac%20OS%20X)%20AppleWebKit%2F600.1.4%20(KHTML%2C%20like%20Gecko)%20Version%2F8.0%20Mobile%2F12D436%20Safari%2F600.1.4
	// ifa=FC0F3445-0FCE-40EE-8646-3CA8BB2663EA
	// ip=208.185.33.66
	// lmt=0
	// w=480
	// h=320
	// protocols=2,3,5,6
	// bundle=com.midasplayer.apps.candycrushsaga
	// tp=s_openx
	// tpv=1.0
	// consent=1
	// gdpr=1
	// pchain=12345

	inmobiurl := "http://api.w.inmobi.com/showad/v3/vast?protocols=2,3,4,6&plid=" + url.QueryEscape(impExt.Plc)
	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			inmobiurl = inmobiurl + "&ua=" + url.QueryEscape(request.Device.UA)
		}
		if len(request.Device.IFA) > 0 {
			inmobiurl = inmobiurl + "&ifa=" + url.QueryEscape(request.Device.IFA)
		}
		if len(request.Device.IP) > 0 {
			inmobiurl = inmobiurl + "&ip=" + url.QueryEscape(request.Device.IP)
		}
		if len(request.Device.IPv6) > 0 {
			inmobiurl = inmobiurl + "&ipv6=" + url.QueryEscape(request.Device.IPv6)
		}
		if request.Device.Lmt != nil {
			inmobiurl = inmobiurl + fmt.Sprintf("&lmt=%d", *(request.Device.Lmt))
		}
		if request.Device.H > 0 {
			inmobiurl = inmobiurl + fmt.Sprintf("&h=%d", request.Device.H)
		}
		if request.Device.W > 0 {
			inmobiurl = inmobiurl + fmt.Sprintf("&w=%d", request.Device.W)
		}
	}
	if request.App != nil {
		if len(request.App.Bundle) > 0 {
			inmobiurl = inmobiurl + "&bundle=" + url.QueryEscape(request.App.Bundle)
		}
	}

	adapterReq := adapters.RequestData{
		Method: "GET",
		Uri:    inmobiurl,
	}

	var adapterRequests []*adapters.RequestData
	adapterRequests = append(adapterRequests, &adapterReq)

	return adapterRequests, errors
}

func (adapter *AdcelInmobiAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	bidResp.ID = internalRequest.ID
	bidResp.BidID = uuid.New().String()
	bidResp.SeatBid = make([]openrtb.SeatBid, 1)
	bidResp.SeatBid[0].Seat = "1"
	bidResp.SeatBid[0].Bid = make([]openrtb.Bid, 1)
	bidResp.SeatBid[0].Bid[0].ID = "1"
	bidResp.SeatBid[0].Bid[0].AdM = fmt.Sprintf("%s", response.Body)
	bidResp.SeatBid[0].Bid[0].ImpID = internalRequest.Imp[0].ID
	bidResp.SeatBid[0].Bid[0].Price = internalRequest.Imp[0].BidFloor

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
		Bid:     &bidResp.SeatBid[0].Bid[0],
		BidType: getMediaTypeForImp(bidResp.SeatBid[0].Bid[0].ImpID, internalRequest.Imp),
	})

	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}

// Builder builds a new instance of the Adcel adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &AdcelInmobiAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
