package adcel

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
)

type AdcelAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

func (adapter *AdcelAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	errors := make([]error, 0, len(request.Imp))

	if request.Imp == nil || len(request.Imp) == 0 {
		errors = append(errors, &errortypes.BadInput{
			Message: fmt.Sprintf("Invalid BidRequest. No valid imp."),
		})
		return nil, errors
	}

	// Adcel uses different sspid parameters for banner and video.
	imps := make([]openrtb.Imp, 0)
	for _, imp := range request.Imp {

		if imp.Audio != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Ignoring imp id=%s, invalid MediaType. Adcel only supports Banner, Video and Native.", imp.ID),
			})
			continue
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding extImpBidder, err: %s.", imp.ID, err),
			})
			continue
		}
		impExt := openrtb_ext.ExtImpAdcel{}
		err := json.Unmarshal(bidderExt.Bidder, &impExt)
		if err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding impExt, err: %s.", imp.ID, err),
			})
			continue
		}
		imps = append(imps, imp)
	}

	var adapterRequests []*adapters.RequestData

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	if len(imps) > 0 {
		// Make a copy as we don't want to change the original request
		reqCopy := *request
		reqCopy.Imp = imps
		reqJSON, err := json.Marshal(reqCopy)
		if err != nil {
			errors = append(errors, err)
			return nil, errors
		}
		adapterReq := adapters.RequestData{
			Method:  "POST",
			Uri:     adapter.URI,
			Body:    reqJSON,
			Headers: headers,
		}
		adapterRequests = append(adapterRequests, &adapterReq)
	}

	return adapterRequests, errors
}

func (adapter *AdcelAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
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

func NewAdcelBidder(client *http.Client, endpoint string) *AdcelAdapter {
	adapter := &adapters.HTTPAdapter{Client: client}
	return &AdcelAdapter{
		http: adapter,
		URI:  endpoint,
	}
}
