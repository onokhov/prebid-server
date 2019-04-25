package engagebdr

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	//"regexp"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
)


const SSPID_TEST_BANNER = "10589"
const SSPID_TEST_VIDEO = "10592"

type EngageBDRAdapter struct {
	http    *adapters.HTTPAdapter
	URI     string
	testing bool
}

func (adapter *EngageBDRAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {

	errors := make([]error, 0)

	var bannerImps []openrtb.Imp
	var videoImps []openrtb.Imp

	for _, imp := range request.Imp {
		// EngageBDR uses different sspid parameters for banner and video.
		if imp.Banner != nil {
			bannerImps = append(bannerImps, imp)
		} else if imp.Video != nil {
			videoImps = append(videoImps, imp)
		}
	}

	var adapterRequests []*adapters.RequestData
	// Make a copy as we don't want to change the original request

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	if len(bannerImps) > 0 {
		reqCopy := *request
		reqCopy.Imp = bannerImps
		reqJSON, err := json.Marshal(reqCopy)
		if err != nil {
			errors = append(errors, err)
			return nil, errors
		}
		sspidBanner := SSPID_TEST_BANNER
		adapterReq := adapters.RequestData{
			Method: "POST",
			Uri:     adapter.URI + "?sspid=" + sspidBanner,
			Body:    reqJSON,
			Headers: headers,
		}
		adapterRequests = append(adapterRequests, &adapterReq)
	}

	if len(videoImps) > 0 {
		reqCopy := *request
		reqCopy.Imp = videoImps
		reqJSON, err := json.Marshal(reqCopy)
		if err != nil {
			errors = append(errors, err)
			return nil, errors
		}
		sspidVideo := SSPID_TEST_VIDEO
		adapterReq := adapters.RequestData{
			Method: "POST",
			Uri:     adapter.URI + "?sspid=" + sspidVideo,
			Body:    reqJSON,
			Headers: headers,
		}
		adapterRequests = append(adapterRequests, &adapterReq)
	}

	if len(adapterRequests) == 0 {
		errors = append(errors, &errortypes.BadInput{ Message: fmt.Sprintf("No imps present") })
		return nil, errors
	}

	return adapterRequests, errors
}

func (adapter *EngageBDRAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}

func NewEngageBDRBidder(client *http.Client, endpoint string) *EngageBDRAdapter {
	adapter := &adapters.HTTPAdapter{Client: client}
	return &EngageBDRAdapter{
		http:    adapter,
		URI:     endpoint,
		testing: false,
	}
}
