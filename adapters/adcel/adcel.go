package adcel

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type AdcelAdapter struct {
	endPoint string
}

// Builder builds a new instance of the Adcel adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &AdcelAdapter{
		endPoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *AdcelAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the request",
		}}
	}

	if err := preprocess(&request.Imp[0]); err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	reqJson, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endPoint,
		Body:    reqJson,
		Headers: headers,
	}}, errs
}
func preprocess(imp *openrtb2.Imp) error {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var adcelExt openrtb_ext.ExtImpAdcel
	if err := json.Unmarshal(bidderExt.Bidder, &adcelExt); err != nil {
		return &errortypes.BadInput{Message: "bad Adcel bidder ext"}
	}

	if adcelExt.Sspid == "" {
		return &errortypes.BadInput{Message: "'sspid' is a required attribute for Adcel's bidder ext"}
	}

	if imp.Banner != nil {
		banner := *imp.Banner
		imp.Banner = &banner
		if (banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0) && len(banner.Format) > 0 {
			format := banner.Format[0]
			banner.W = &format.W
			banner.H = &format.H
		}
	}

	return nil
}

func (a *AdcelAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected http status code: %d", response.StatusCode),
		}}
	}

	var serverBidResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &serverBidResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range serverBidResponse.SeatBid {
		for i := range sb.Bid {
			mediaType := getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}

	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			break
		}
	}
	return mediaType
}
