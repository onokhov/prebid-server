package adcel

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

func NewAdcelSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adcel", 0, temp, adapters.SyncTypeIframe)
}
