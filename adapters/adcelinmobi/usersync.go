package adcelinmobi

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

func NewAdcelInmobiSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adcelinmobi", 0, temp, adapters.SyncTypeIframe)
}
