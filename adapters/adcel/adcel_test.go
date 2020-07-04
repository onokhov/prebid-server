package adcel

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"net/http"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adceltest", NewAdcelBidder(new(http.Client), "http://dsp.adcel.co/hb"))
}
